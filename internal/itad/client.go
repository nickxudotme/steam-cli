package itad

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"steam-cli/internal/steam"
)

const steamShopID = 61

func SteamShopID() int {
	return steamShopID
}

type Client struct {
	Key        string
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(key string, timeout time.Duration) *Client {
	return &Client{
		Key:     strings.TrimSpace(key),
		BaseURL: "https://api.isthereanydeal.com",
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) LookupByAppID(appid int) (*Game, error) {
	endpoint := c.BaseURL + "/games/lookup/v1"
	var payload struct {
		Found bool `json:"found"`
		Game  Game `json:"game"`
	}
	if err := c.getJSON(endpoint, url.Values{"appid": {strconv.Itoa(appid)}}, &payload); err != nil {
		return nil, err
	}
	if !payload.Found || payload.Game.ID == "" {
		return nil, &steam.Error{
			Code:    steam.CodeNotFound,
			Message: fmt.Sprintf("no IsThereAnyDeal game found for appid %d", appid),
		}
	}
	payload.Game.AppID = appid
	return &payload.Game, nil
}

func (c *Client) SummaryByAppID(appid int, country string) (*Summary, error) {
	game, err := c.LookupByAppID(appid)
	if err != nil {
		return nil, err
	}

	summary := &Summary{Game: *game}
	var (
		overview   *Overview
		historyLow *HistoryLow
		steamLow   *StoreLow
		bundles    []Bundle
		errMu      sync.Mutex
		firstErr   error
	)
	setErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		errMu.Unlock()
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		resp, err := c.Overview(country, []string{game.ID}, nil)
		if err != nil {
			setErr(err)
			return
		}
		if len(resp.Prices) > 0 {
			overview = &resp.Prices[0]
		}
		bundles = resp.Bundles
	}()
	go func() {
		defer wg.Done()
		items, err := c.HistoryLow(country, []string{game.ID})
		if err != nil {
			setErr(err)
			return
		}
		if len(items) > 0 {
			historyLow = &items[0]
		}
	}()
	go func() {
		defer wg.Done()
		items, err := c.StoreLow(country, []string{game.ID}, []int{steamShopID})
		if err != nil {
			setErr(err)
			return
		}
		if len(items) > 0 {
			steamLow = &items[0]
		}
	}()
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	summary.Overview = overview
	summary.HistoryLow = historyLow
	summary.SteamLow = steamLow
	summary.Bundles = bundles
	return summary, nil
}

func (c *Client) Overview(country string, gids []string, shops []int) (*OverviewResponse, error) {
	endpoint := c.BaseURL + "/games/overview/v2"
	query := url.Values{"country": {country}}
	if len(shops) > 0 {
		query.Set("shops", joinShopIDs(shops))
	}
	var payload OverviewResponse
	if err := c.postJSON(endpoint, query, gids, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

type OverviewResponse struct {
	Prices  []Overview `json:"prices"`
	Bundles []Bundle   `json:"bundles"`
}

func (c *Client) HistoryLow(country string, gids []string) ([]HistoryLow, error) {
	endpoint := c.BaseURL + "/games/historylow/v1"
	query := url.Values{"country": {country}}
	var payload []HistoryLow
	if err := c.postJSON(endpoint, query, gids, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *Client) StoreLow(country string, gids []string, shops []int) ([]StoreLow, error) {
	endpoint := c.BaseURL + "/games/storelow/v2"
	query := url.Values{"country": {country}}
	if len(shops) > 0 {
		query.Set("shops", joinShopIDs(shops))
	}
	var payload []StoreLow
	if err := c.postJSON(endpoint, query, gids, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *Client) History(gid, country string, shops []int, since string) ([]HistoryEntry, error) {
	endpoint := c.BaseURL + "/games/history/v2"
	query := url.Values{
		"id":      {gid},
		"country": {country},
	}
	if len(shops) > 0 {
		query.Set("shops", joinShopIDs(shops))
	}
	if since != "" {
		query.Set("since", since)
	}
	var payload []HistoryEntry
	if err := c.getJSON(endpoint, query, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *Client) Shops() ([]Shop, error) {
	endpoint := c.BaseURL + "/service/shops/v1"
	var payload []Shop
	if err := c.getJSON(endpoint, nil, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *Client) getJSON(endpoint string, query url.Values, out any) error {
	reqURL := endpoint
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return &steam.Error{Code: steam.CodeUnknown, Message: fmt.Sprintf("build request for %s", endpoint), Cause: err}
	}
	return c.doJSON(req, endpoint, out)
}

func (c *Client) postJSON(endpoint string, query url.Values, body any, out any) error {
	reqURL := endpoint
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return &steam.Error{Code: steam.CodeUnknown, Message: fmt.Sprintf("encode request for %s", endpoint), Cause: err}
	}
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(raw))
	if err != nil {
		return &steam.Error{Code: steam.CodeUnknown, Message: fmt.Sprintf("build request for %s", endpoint), Cause: err}
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doJSON(req, endpoint, out)
}

func (c *Client) doJSON(req *http.Request, endpoint string, out any) error {
	if err := c.ensureKey(); err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("ITAD-API-Key", c.Key)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return wrapNetwork(err, endpoint)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return wrapHTTPStatus(resp.StatusCode, endpoint, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &steam.Error{
			Code:    steam.CodeSourceChanged,
			Message: fmt.Sprintf("invalid response from %s", endpoint),
			Cause:   err,
		}
	}
	return nil
}

func (c *Client) ensureKey() error {
	if c.Key != "" {
		return nil
	}
	return &steam.Error{
		Code:    steam.CodeAccessDenied,
		Message: "missing IsThereAnyDeal API key; set --itad-key or STEAM_CLI_ITAD_KEY",
	}
}

func wrapHTTPStatus(status int, endpoint, detail string) error {
	message := fmt.Sprintf("HTTP %d from %s", status, endpoint)
	if detail != "" {
		message += ": " + detail
	}
	code := steam.CodeUnknown
	switch status {
	case 400:
		code = steam.CodeInvalidInput
	case 401, 403:
		code = steam.CodeAccessDenied
	case 404:
		code = steam.CodeNotFound
	case 429:
		code = steam.CodeRateLimited
	}
	return &steam.Error{Code: code, Message: message}
}

func wrapNetwork(err error, endpoint string) error {
	type timeoutError interface{ Timeout() bool }
	var te timeoutError
	if errors.As(err, &te) && te.Timeout() {
		return &steam.Error{
			Code:    steam.CodeNetworkTimeout,
			Message: fmt.Sprintf("network timeout for %s", endpoint),
			Cause:   err,
		}
	}
	return &steam.Error{
		Code:    steam.CodeUnknown,
		Message: fmt.Sprintf("request failed for %s", endpoint),
		Cause:   err,
	}
}

func joinShopIDs(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.Itoa(id))
	}
	return strings.Join(out, ",")
}
