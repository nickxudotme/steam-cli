package steam

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Endpoints groups the base URLs the client talks to. Tests inject
// httptest.NewServer URLs here; production gets DefaultEndpoints.
type Endpoints struct {
	Store     string
	API       string
	Community string
	CDN       string
}

// DefaultEndpoints returns Steam's live public hosts.
func DefaultEndpoints() Endpoints {
	return Endpoints{
		Store:     "https://store.steampowered.com",
		API:       "https://api.steampowered.com",
		Community: "https://steamcommunity.com",
		CDN:       "https://cdn.akamai.steamstatic.com",
	}
}

// userAgent is exported so tests and the doctor command share one identity.
const UserAgent = "steam-cli/" + Version

// Version is updated alongside cmd.version. It lives here so the steam package
// has no dependency on cmd.
const Version = "0.1.0"

// Client is a Steam HTTP client. Construct via NewClient.
//
// Concurrency: the Client and its Cache are safe for concurrent use.
type Client struct {
	CC         string
	Lang       string
	HTTPClient *http.Client
	Endpoints  Endpoints

	// Cache is shared across regional clients via DefaultCache by default.
	// Tests can override with NewCache().
	Cache *Cache

	// MinInterval throttles consecutive requests from this Client. Zero = off.
	// Useful for batched per-app lookups (e.g. wishlist with --no-details=false)
	// to avoid 429s without hard-coding sleeps in business code.
	MinInterval time.Duration

	// RetryLogger is called before sleeping for a retry. CLI callers use this
	// for stderr diagnostics; library callers can leave it nil for silence.
	RetryLogger func(RetryEvent)

	rateMu  sync.Mutex
	lastReq time.Time
}

// RetryEvent describes a retry wait that is about to happen.
type RetryEvent struct {
	URL         string
	Status      int
	Err         error
	Attempt     int
	MaxAttempts int
	Delay       time.Duration
	RetryAfter  bool
}

// NewClient returns a Client ready to talk to the live Steam endpoints.
func NewClient(cc, lang string, timeout time.Duration) *Client {
	return &Client{
		CC:   cc,
		Lang: lang,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		Endpoints: DefaultEndpoints(),
		Cache:     DefaultCache,
	}
}

func (c *Client) AppBundle(appid int, newsCount int) (*AppBundle, error) {
	details, err := c.AppDetails(appid)
	if err != nil {
		return nil, err
	}

	bundle := &AppBundle{
		AppID:   appid,
		Details: details,
	}

	var (
		storeItem *StoreItem
		reviews   *ReviewSummary
		players   *int
		news      []NewsItem
	)
	var warnMu sync.Mutex
	var warnings []string
	addWarn := func(label string, err error) {
		if err == nil {
			return
		}
		warnMu.Lock()
		warnings = append(warnings, label+": "+err.Error())
		warnMu.Unlock()
	}

	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		item, err := c.StoreItem(appid)
		if err != nil {
			addWarn("store_item", err)
			return
		}
		storeItem = item
	}()
	go func() {
		defer wg.Done()
		r, err := c.AppReviews(appid)
		if err != nil {
			addWarn("reviews", err)
			return
		}
		reviews = r
	}()
	go func() {
		defer wg.Done()
		p, err := c.CurrentPlayers(appid)
		if err != nil {
			addWarn("current_players", err)
			return
		}
		players = p
	}()
	go func() {
		defer wg.Done()
		n, err := c.News(appid, newsCount)
		if err != nil {
			addWarn("news", err)
			return
		}
		news = n
	}()
	wg.Wait()

	bundle.StoreItem = storeItem
	bundle.Reviews = reviews
	bundle.CurrentPlayers = players
	bundle.News = news
	bundle.Warnings = warnings
	return bundle, nil
}

func (c *Client) AppDetails(appid int) (*AppDetails, error) {
	cacheKey := fmt.Sprintf("%s:%s:%d", c.CC, c.Lang, appid)
	if cached, ok := c.Cache.GetAppDetails(cacheKey); ok {
		return cached, nil
	}

	endpoint := c.Endpoints.Store + "/api/appdetails"
	var payload map[string]struct {
		Success bool        `json:"success"`
		Data    *AppDetails `json:"data"`
	}
	if err := c.getJSON(endpoint, url.Values{
		"appids": {strconv.Itoa(appid)},
		"cc":     {c.CC},
		"l":      {c.Lang},
	}, &payload); err != nil {
		return nil, err
	}

	wrapped, ok := payload[strconv.Itoa(appid)]
	if !ok || !wrapped.Success || wrapped.Data == nil {
		return nil, newNotFound("no public app details for appid %d", appid)
	}
	c.Cache.SetAppDetails(cacheKey, wrapped.Data)
	return wrapped.Data, nil
}

func (c *Client) StoreItem(appid int) (*StoreItem, error) {
	cacheKey := fmt.Sprintf("%s:%s:%d", c.CC, c.Lang, appid)
	if cached, ok := c.Cache.GetStoreItem(cacheKey); ok {
		return cached, nil
	}
	items, err := c.StoreItems([]int{appid})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, newNotFound("no store browse item for appid %d", appid)
	}
	item := &items[0]
	c.Cache.SetStoreItem(cacheKey, item)
	return item, nil
}

func (c *Client) StoreItems(appids []int) ([]StoreItem, error) {
	if len(appids) == 0 {
		return nil, nil
	}
	input := storeBrowseRequest{
		Context: storeBrowseContext{
			Language:    c.Lang,
			CountryCode: c.CC,
		},
		DataRequest: fullStoreItemDataRequest(),
	}
	for _, appid := range appids {
		input.IDs = append(input.IDs, storeItemID{AppID: appid})
	}

	var payload struct {
		Response struct {
			StoreItems []StoreItem `json:"store_items"`
		} `json:"response"`
	}
	if err := c.storeJSON(c.Endpoints.API+"/IStoreBrowseService/GetItems/v1/", input, &payload); err != nil {
		return nil, err
	}
	return payload.Response.StoreItems, nil
}

func (c *Client) DLC(appid int) (*DLCResult, error) {
	details, err := c.AppDetails(appid)
	if err != nil {
		return nil, err
	}
	result := &DLCResult{
		AppID: appid,
		Name:  details.Name,
		Total: len(details.DLC),
	}
	if len(details.DLC) == 0 {
		return result, nil
	}
	items, err := c.StoreItems(details.DLC)
	if err != nil {
		return nil, err
	}
	result.Items = items
	result.Total = len(items)
	return result, nil
}

func (c *Client) Similar(appid int, count int) (*SimilarResult, error) {
	if count <= 0 {
		count = 10
	}
	input := storeSimilarRequest{
		ItemID: storeItemID{AppID: appid},
		Context: storeBrowseContext{
			Language:    c.Lang,
			CountryCode: c.CC,
		},
		DataRequest: fullStoreItemDataRequest(),
		Count:       count,
	}
	var payload struct {
		Response struct {
			StoreItems []StoreItem `json:"store_items"`
		} `json:"response"`
	}
	if err := c.storeJSON(c.Endpoints.API+"/IStoreQueryService/MoreLikeThis/v1/", input, &payload); err != nil {
		return nil, err
	}
	return &SimilarResult{
		AppID: appid,
		Items: payload.Response.StoreItems,
		Total: len(payload.Response.StoreItems),
	}, nil
}

func (c *Client) AppReviews(appid int) (*ReviewSummary, error) {
	payload, err := c.Reviews(appid, ReviewQuery{Count: 0, Language: c.Lang, PurchaseType: "all"})
	if err != nil {
		return nil, err
	}
	return &payload.QuerySummary, nil
}

type ReviewQuery struct {
	Count        int
	Language     string
	Filter       string
	ReviewType   string
	PurchaseType string
	Cursor       string
}

func (c *Client) Reviews(appid int, query ReviewQuery) (*ReviewResponse, error) {
	if query.Language == "" {
		query.Language = c.Lang
	}
	if query.Filter == "" {
		query.Filter = "recent"
	}
	if query.ReviewType == "" {
		query.ReviewType = "all"
	}
	if query.PurchaseType == "" {
		query.PurchaseType = "all"
	}
	params := url.Values{
		"json":          {"1"},
		"language":      {query.Language},
		"filter":        {query.Filter},
		"review_type":   {query.ReviewType},
		"purchase_type": {query.PurchaseType},
		"num_per_page":  {strconv.Itoa(query.Count)},
	}
	if query.Cursor != "" {
		params.Set("cursor", query.Cursor)
	}
	var payload ReviewResponse
	if err := c.getJSON(fmt.Sprintf("%s/appreviews/%d", c.Endpoints.Store, appid), params, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func (c *Client) CurrentPlayers(appid int) (*int, error) {
	var payload struct {
		Response struct {
			Result      int `json:"result"`
			PlayerCount int `json:"player_count"`
		} `json:"response"`
	}
	endpoint := c.Endpoints.API + "/ISteamUserStats/GetNumberOfCurrentPlayers/v1/"
	if err := c.getJSON(endpoint, url.Values{
		"appid": {strconv.Itoa(appid)},
	}, &payload); err != nil {
		return nil, err
	}
	if payload.Response.Result != 1 {
		return nil, newNotFound("player count unavailable for appid %d", appid)
	}
	count := payload.Response.PlayerCount
	return &count, nil
}

func (c *Client) News(appid int, count int) ([]NewsItem, error) {
	var payload struct {
		AppNews struct {
			NewsItems []NewsItem `json:"newsitems"`
		} `json:"appnews"`
	}
	endpoint := c.Endpoints.API + "/ISteamNews/GetNewsForApp/v2/"
	if err := c.getJSON(endpoint, url.Values{
		"appid":     {strconv.Itoa(appid)},
		"count":     {strconv.Itoa(count)},
		"maxlength": {"300"},
		"format":    {"json"},
		"feeds":     {"steam_community_announcements,steam_community_events"},
	}, &payload); err != nil {
		return nil, err
	}
	return payload.AppNews.NewsItems, nil
}

func (c *Client) Search(term string, count int) ([]SearchItem, error) {
	var payload struct {
		Items []SearchItem `json:"items"`
	}
	if err := c.getJSON(c.Endpoints.Store+"/api/storesearch", url.Values{
		"term":          {term},
		"cc":            {c.CC},
		"l":             {c.Lang},
		"category1":     {"998"},
		"supportedlang": {c.Lang},
		"ndl":           {"1"},
	}, &payload); err != nil {
		return nil, err
	}
	if count > 0 && count < len(payload.Items) {
		return payload.Items[:count], nil
	}
	return payload.Items, nil
}

func (c *Client) LiveLanguages() ([]LanguageOption, error) {
	lang := strings.TrimSpace(c.Lang)
	if lang == "" {
		lang = "english"
	}
	body, err := c.GetText(c.Endpoints.Store+"/", url.Values{"l": {lang}})
	if err != nil {
		return nil, err
	}
	return ParseSteamStoreLanguagesHTML(body), nil
}

func (c *Client) ProbeRegions(appid int, regions []RegionOption) (*RegionProbeResult, error) {
	if appid <= 0 {
		return nil, newInvalidInput("appid must be positive")
	}
	result := &RegionProbeResult{
		AppID:  appid,
		Source: c.Endpoints.Store + "/api/appdetails",
	}
	for i, region := range regions {
		if i > 0 {
			time.Sleep(150 * time.Millisecond)
		}
		probe := RegionProbe{
			Code: region.Code,
			Name: region.Name,
		}
		// Don't copy *c — it embeds a sync.Mutex (rate limiter). Build a
		// dedicated Client per region that shares the same Cache and
		// HTTPClient/Endpoints, so DefaultCache hits still work across regions.
		regionClient := &Client{
			CC:          strings.ToUpper(region.Code),
			Lang:        c.Lang,
			HTTPClient:  c.HTTPClient,
			Endpoints:   c.Endpoints,
			Cache:       c.Cache,
			MinInterval: c.MinInterval,
			RetryLogger: c.RetryLogger,
		}
		details, err := regionClient.AppDetails(appid)
		if err != nil {
			probe.Error = err.Error()
			result.Regions = append(result.Regions, probe)
			continue
		}
		probe.Available = true
		if details.PriceOverview != nil {
			probe.Currency = details.PriceOverview.Currency
			probe.FinalFormatted = details.PriceOverview.FinalFormatted
		}
		result.Regions = append(result.Regions, probe)
	}
	return result, nil
}

func (c *Client) GlobalAchievements(appid int) ([]GlobalAchievement, error) {
	var payload struct {
		AchievementPercentages struct {
			Achievements []struct {
				Name    string `json:"name"`
				Percent string `json:"percent"`
			} `json:"achievements"`
		} `json:"achievementpercentages"`
	}
	endpoint := c.Endpoints.API + "/ISteamUserStats/GetGlobalAchievementPercentagesForApp/v2/"
	if err := c.getJSON(endpoint, url.Values{
		"gameid": {strconv.Itoa(appid)},
	}, &payload); err != nil {
		return nil, err
	}
	achievements := make([]GlobalAchievement, 0, len(payload.AchievementPercentages.Achievements))
	for _, item := range payload.AchievementPercentages.Achievements {
		var percent float64
		fmt.Sscanf(item.Percent, "%f", &percent)
		achievements = append(achievements, GlobalAchievement{Name: item.Name, Percent: percent})
	}
	return achievements, nil
}

func (c *Client) StoreResults(filter string, count int) ([]StoreResult, error) {
	return c.StoreResultsQuery(StoreResultsQuery{Filter: filter, Count: count})
}

func (c *Client) StoreResultsQuery(query StoreResultsQuery) ([]StoreResult, error) {
	filter := query.Filter
	if filter == "" {
		filter = "specials"
	}
	originalFilter := filter
	anyConditions := append([]StoreResultCondition(nil), query.Any...)
	allConditions := append([]StoreResultCondition(nil), query.All...)
	switch filter {
	case "discountedtopsellers":
		filter = "topsellers"
		allConditions = append(allConditions, StoreResultConditionDiscounted)
	case "preordertopsellers":
		filter = "topsellers"
		allConditions = append(allConditions, StoreResultConditionPreorder)
	}

	requestCount := query.Count
	if hasStoreResultConditions(anyConditions, allConditions) && requestCount > 0 && requestCount < 50 {
		requestCount = 50
	}
	params := url.Values{
		"query":        {""},
		"start":        {"0"},
		"count":        {strconv.Itoa(requestCount)},
		"dynamic_data": {""},
		"infinite":     {"1"},
		"cc":           {c.CC},
		"l":            {c.Lang},
	}
	if originalFilter == "discountedtopsellers" {
		params.Set("filter", "topsellers")
		params.Set("specials", "1")
	} else {
		switch filter {
		case "specials":
			params.Set("specials", "1")
		case "topsellers", "globaltopsellers", "new", "comingsoon":
			params.Set("filter", filter)
		default:
			params.Set("filter", filter)
		}
	}
	var payload struct {
		Success     int    `json:"success"`
		ResultsHTML string `json:"results_html"`
		TotalCount  int    `json:"total_count"`
		Start       int    `json:"start"`
	}
	if err := c.getJSON(c.Endpoints.Store+"/search/results/", params, &payload); err != nil {
		return nil, err
	}
	results := ParseStoreResultsHTML(payload.ResultsHTML)
	storeItems := map[int]StoreItem{}
	if hasStoreResultConditions(anyConditions, allConditions) {
		filtered, items, err := c.filterStoreResults(results, anyConditions, allConditions)
		if err != nil {
			return nil, err
		}
		results = filtered
		storeItems = items
	}
	if query.Count > 0 && query.Count < len(results) {
		results = results[:query.Count]
	}
	if err := c.enrichStoreResultData(results, storeItems); err != nil {
		return nil, err
	}
	return results, nil
}

func hasStoreResultConditions(groups ...[]StoreResultCondition) bool {
	for _, group := range groups {
		if len(group) > 0 {
			return true
		}
	}
	return false
}

func (c *Client) filterStoreResults(results []StoreResult, anyConditions, allConditions []StoreResultCondition) ([]StoreResult, map[int]StoreItem, error) {
	storeItems := map[int]StoreItem{}
	if storeResultConditionsNeedStoreItems(anyConditions, allConditions) {
		appids := make([]int, 0, len(results))
		for _, result := range results {
			appids = append(appids, result.AppID)
		}
		items, err := c.StoreItems(appids)
		if err != nil {
			return nil, nil, err
		}
		for _, item := range items {
			storeItems[item.AppID] = item
		}
	}
	filtered := results[:0]
	for _, result := range results {
		item, hasItem := storeItems[result.AppID]
		if storeResultMatches(result, item, hasItem, anyConditions, allConditions) {
			filtered = append(filtered, result)
		}
	}
	return filtered, storeItems, nil
}

func storeResultConditionsNeedStoreItems(groups ...[]StoreResultCondition) bool {
	for _, group := range groups {
		for _, condition := range group {
			if condition == StoreResultConditionPreorder {
				return true
			}
		}
	}
	return false
}

func storeResultMatches(result StoreResult, item StoreItem, hasItem bool, anyConditions, allConditions []StoreResultCondition) bool {
	for _, condition := range allConditions {
		if !storeResultMatchesCondition(result, item, hasItem, condition) {
			return false
		}
	}
	if len(anyConditions) == 0 {
		return true
	}
	for _, condition := range anyConditions {
		if storeResultMatchesCondition(result, item, hasItem, condition) {
			return true
		}
	}
	return false
}

func storeResultMatchesCondition(result StoreResult, item StoreItem, hasItem bool, condition StoreResultCondition) bool {
	switch condition {
	case StoreResultConditionDiscounted:
		return isDiscountedStoreResult(result)
	case StoreResultConditionPreorder:
		return hasItem && isPreorderStoreItem(item)
	default:
		return false
	}
}

func isPreorderStoreItem(item StoreItem) bool {
	comingSoon := item.IsComingSoon
	if item.Release != nil && item.Release.IsComingSoon {
		comingSoon = true
	}
	if !comingSoon || item.IsFree {
		return false
	}
	if item.BestPurchaseOption != nil && item.BestPurchaseOption.FinalPriceInCents > 0 {
		return true
	}
	for _, option := range item.PurchaseOptions {
		if option.FinalPriceInCents > 0 {
			return true
		}
	}
	return false
}

func isDiscountedStoreResult(result StoreResult) bool {
	return result.Discount != "" && result.Discount != "-"
}

func (c *Client) enrichStoreResultData(results []StoreResult, storeItems map[int]StoreItem) error {
	if storeItems == nil {
		storeItems = map[int]StoreItem{}
	}
	needed := make([]int, 0, len(results))
	seen := map[int]bool{}
	for _, result := range results {
		if _, ok := storeItems[result.AppID]; ok || seen[result.AppID] {
			continue
		}
		needed = append(needed, result.AppID)
		seen[result.AppID] = true
	}
	if len(needed) > 0 {
		items, err := c.StoreItems(needed)
		if err != nil {
			return err
		}
		for _, item := range items {
			storeItems[item.AppID] = item
		}
	}
	for index, result := range results {
		if item, ok := storeItems[result.AppID]; ok {
			results[index].setReleaseFromStoreItem(item)
			if isDiscountedStoreResult(result) {
				results[index].DiscountEnd = latestStoreItemDiscountEnd(item)
			}
		}
	}
	return nil
}

func (result *StoreResult) setReleaseFromStoreItem(item StoreItem) {
	if item.Release == nil {
		return
	}
	timestamp := item.Release.SteamReleaseDate
	if timestamp == 0 {
		timestamp = item.Release.OriginalSteamReleaseDate
	}
	if timestamp == 0 {
		timestamp = item.Release.OriginalReleaseDate
	}
	if timestamp == 0 {
		return
	}
	result.ReleaseTime = timestamp
}

func latestStoreItemDiscountEnd(item StoreItem) int64 {
	var latest int64
	if item.BestPurchaseOption != nil {
		latest = latestActiveDiscountEnd(item.BestPurchaseOption.ActiveDiscounts)
	}
	for _, option := range item.PurchaseOptions {
		if end := latestActiveDiscountEnd(option.ActiveDiscounts); end > latest {
			latest = end
		}
	}
	return latest
}

func latestActiveDiscountEnd(discounts []ActiveDiscount) int64 {
	var latest int64
	for _, discount := range discounts {
		if discount.DiscountEndDate > latest {
			latest = discount.DiscountEndDate
		}
	}
	return latest
}

func (c *Client) UserProfile(input string) (*UserProfile, error) {
	path, err := profilePath(input)
	if err != nil {
		return nil, err
	}
	if cached, ok := c.Cache.GetProfile(path); ok {
		return cached, nil
	}

	body, err := c.GetText(c.Endpoints.Community+"/"+path+"/", url.Values{"xml": {"1"}})
	if err != nil {
		return nil, err
	}
	var profile UserProfile
	if err := xml.Unmarshal([]byte(body), &profile); err != nil {
		return nil, newSourceChanged(err, "Steam Community profile XML")
	}
	if profile.SteamID64 == "" {
		return nil, newNotFound("no public Steam profile found for %q", input)
	}
	c.Cache.SetProfile(path, &profile)
	return &profile, nil
}

type wishlistRawItem struct {
	AppID     int   `json:"appid"`
	Priority  int   `json:"priority"`
	DateAdded int64 `json:"date_added"`
}

func (c *Client) Wishlist(input string, offset int, count int) (*Wishlist, error) {
	return c.WishlistWithDetails(input, offset, count, true)
}

func (c *Client) WishlistWithDetails(input string, offset int, count int, includeDetails bool) (*Wishlist, error) {
	steamID64, err := c.steamID64(input)
	if err != nil {
		return nil, err
	}

	rawItems, ok := c.Cache.GetWishlist(steamID64)
	if !ok {
		var payload struct {
			Response struct {
				Items []wishlistRawItem `json:"items"`
			} `json:"response"`
		}
		if err := c.getJSON(c.Endpoints.API+"/IWishlistService/GetWishlist/v1/", url.Values{
			"steamid": {steamID64},
		}, &payload); err != nil {
			return nil, err
		}
		rawItems = payload.Response.Items
		c.Cache.SetWishlist(steamID64, rawItems)
	}
	total := len(rawItems)
	if total == 0 {
		return nil, newPrivacyRestricted("no public wishlist items found for %s; the wishlist may be private, friends-only, or empty", steamID64)
	}
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := total
	if count > 0 && offset+count < end {
		end = offset + count
	}

	items := make([]WishlistItem, 0, end-offset)
	// Use the Client's configured MinInterval to space per-app appdetails
	// fetches. Caller can tune via --rate-ms; default 0 means no throttle
	// here, but the cache still suppresses repeated fetches of the same id.
	for _, raw := range rawItems[offset:end] {
		item := WishlistItem{
			AppID:     raw.AppID,
			Priority:  raw.Priority,
			DateAdded: raw.DateAdded,
		}
		if includeDetails {
			details, err := c.AppDetails(raw.AppID)
			if err != nil {
				item.Error = err.Error()
			} else {
				item.Details = details
			}
		}
		items = append(items, item)
	}

	return &Wishlist{
		SteamID64: steamID64,
		Items:     items,
		Total:     total,
		Offset:    offset,
		Count:     len(items),
	}, nil
}

func (c *Client) Media(appid int, probe bool) (*Media, error) {
	details, err := c.AppDetails(appid)
	if err != nil {
		return nil, err
	}
	storeItem, _ := c.StoreItem(appid)
	assets := mediaAssetsFromStore(c.Endpoints.CDN, appid, storeItem)
	if probe {
		for i := range assets {
			status := c.ProbeURL(assets[i].URL)
			available := status >= 200 && status < 400
			assets[i].Status = status
			assets[i].Available = &available
		}
	}

	var achievements []HighlightedAchievement
	if details.Achievements != nil {
		achievements = details.Achievements.Highlighted
	}
	return &Media{
		AppID:            appid,
		Name:             details.Name,
		HeaderImage:      details.HeaderImage,
		CDNAssets:        assets,
		Screenshots:      details.Screenshots,
		Movies:           details.Movies,
		AchievementIcons: achievements,
	}, nil
}

func mediaAssetsFromStore(cdnBase string, appid int, item *StoreItem) []MediaAsset {
	fallback := func(kind, name, filename string) MediaAsset {
		return MediaAsset{
			Kind: kind,
			Name: name,
			URL:  fmt.Sprintf("%s/steam/apps/%d/%s", cdnBase, appid, filename),
		}
	}
	if item == nil || item.Assets == nil || item.Assets.AssetURLFormat == "" {
		return []MediaAsset{
			fallback("header", "header", "header.jpg"),
			fallback("capsule", "capsule_616x353", "capsule_616x353.jpg"),
			fallback("capsule", "capsule_231x87", "capsule_231x87.jpg"),
			fallback("library", "library_600x900", "library_600x900.jpg"),
			fallback("library", "library_hero", "library_hero.jpg"),
			fallback("library", "logo", "logo.png"),
		}
	}
	assets := item.Assets
	candidates := []struct {
		kind     string
		name     string
		filename string
	}{
		{"capsule", "main_capsule", assets.MainCapsule},
		{"capsule", "main_capsule_2x", assets.MainCapsule2x},
		{"capsule", "capsule_231x87", assets.SmallCapsule},
		{"capsule", "capsule_231x87_2x", assets.SmallCapsule2x},
		{"header", "header", assets.Header},
		{"header", "header_2x", assets.Header2x},
		{"capsule", "hero_capsule", assets.HeroCapsule},
		{"capsule", "hero_capsule_2x", assets.HeroCapsule2x},
		{"library", "library_600x900", assets.LibraryCapsule},
		{"library", "library_600x900_2x", assets.LibraryCapsule2x},
		{"library", "library_hero", assets.LibraryHero},
		{"library", "library_hero_2x", assets.LibraryHero2x},
		{"background", "page_background", assets.PageBackground},
		{"background", "raw_page_background", assets.RawPageBackground},
	}
	out := make([]MediaAsset, 0, len(candidates))
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate.filename == "" {
			continue
		}
		url := storeAssetURL(cdnBase, assets.AssetURLFormat, candidate.filename)
		if url == "" || seen[url] {
			continue
		}
		seen[url] = true
		out = append(out, MediaAsset{Kind: candidate.kind, Name: candidate.name, URL: url})
	}
	return out
}

func storeAssetURL(cdnBase, format, filename string) string {
	if format == "" || filename == "" {
		return ""
	}
	path := strings.ReplaceAll(format, "${FILENAME}", filename)
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if strings.HasPrefix(path, "//") {
		return "https:" + path
	}
	if strings.HasPrefix(path, "steam/apps/") {
		return "https://shared.akamai.steamstatic.com/store_item_assets/" + path
	}
	return cdnBase + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) ProbeURL(rawURL string) int {
	req, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func (c *Client) steamID64(input string) (string, error) {
	path, err := profilePath(input)
	if err != nil {
		return "", err
	}
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[0] == "profiles" {
		return parts[1], nil
	}
	profile, err := c.UserProfile(input)
	if err != nil {
		return "", err
	}
	return profile.SteamID64, nil
}

var (
	steamID64Re = regexp.MustCompile(`^\d{17}$`)
	customURLRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

func profilePath(input string) (string, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return "", newInvalidProfileInput("empty Steam user input")
	}
	value = strings.TrimRight(value, "/")
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		host := strings.ToLower(parsed.Host)
		if host != "steamcommunity.com" && host != "www.steamcommunity.com" {
			return "", newInvalidProfileInput("unsupported Steam profile URL %q", input)
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) >= 2 && (parts[0] == "id" || parts[0] == "profiles") {
			return parts[0] + "/" + parts[1], nil
		}
		return "", newInvalidProfileInput("unsupported Steam profile URL %q", input)
	}
	if steamID64Re.MatchString(value) {
		return "profiles/" + value, nil
	}
	if customURLRe.MatchString(value) {
		return "id/" + value, nil
	}
	return "", newInvalidProfileInput("expected SteamID64, custom URL name, or steamcommunity profile URL")
}

// --- HTTP plumbing ----------------------------------------------------------

func (c *Client) getJSON(endpoint string, params url.Values, target any) error {
	return c.getJSONWithUserAgent(endpoint, params, UserAgent, target)
}

func (c *Client) getJSONWithUserAgent(endpoint string, params url.Values, userAgent string, target any) error {
	body, err := c.getText(endpoint, params, userAgent)
	if err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(body), target); err != nil {
		return newSourceChanged(err, endpoint)
	}
	return nil
}

func (c *Client) storeJSON(endpoint string, input any, target any) error {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return c.getJSONWithUserAgent(endpoint, url.Values{
		"input_json": {string(inputJSON)},
	}, "Valve/Steam HTTP Client 1.0", target)
}

// GetText is the public form used by HTML parsers (events, locales).
func (c *Client) GetText(endpoint string, params url.Values) (string, error) {
	return c.getText(endpoint, params, UserAgent)
}

// retryPolicy controls the request-level retry loop.
type retryPolicy struct {
	maxAttempts        int
	retriableStatuses  map[int]bool
	defaultBackoff     time.Duration
	maxRetryAfterDelay time.Duration
}

func defaultRetryPolicy() retryPolicy {
	return retryPolicy{
		maxAttempts:        3,
		retriableStatuses:  map[int]bool{429: true, 502: true, 503: true, 504: true},
		defaultBackoff:     2 * time.Second,
		maxRetryAfterDelay: 30 * time.Second,
	}
}

func (c *Client) getText(endpoint string, params url.Values, userAgent string) (string, error) {
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	policy := defaultRetryPolicy()
	var lastErr error
	for attempt := 0; attempt < policy.maxAttempts; attempt++ {
		c.waitForInterval()

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(userAgent, "Valve/Steam") {
			req.Header.Set("Accept", "application/json,*/*;q=0.8")
		} else {
			req.Header.Set("Accept", "application/json,text/html;q=0.9,*/*;q=0.8")
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt == policy.maxAttempts-1 {
				return "", wrapNetwork(err, endpoint)
			}
			delay := backoffFor(attempt, policy.defaultBackoff)
			c.logRetry(RetryEvent{
				URL:         endpoint,
				Err:         err,
				Attempt:     attempt + 1,
				MaxAttempts: policy.maxAttempts,
				Delay:       delay,
			})
			time.Sleep(delay)
			continue
		}

		// Non-retriable response: read body or surface HTTPError.
		if !policy.retriableStatuses[resp.StatusCode] {
			body, readErr := readAllAndClose(resp.Body)
			if readErr != nil {
				return "", wrapNetwork(readErr, endpoint)
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return "", wrapHTTPStatus(&HTTPError{Status: resp.StatusCode, URL: endpoint})
			}
			return string(body), nil
		}

		// Retriable response (429/5xx). Decide on delay.
		retryDelay := retryAfterDuration(resp.Header.Get("Retry-After"))
		resp.Body.Close()

		if attempt == policy.maxAttempts-1 {
			return "", wrapHTTPStatus(&HTTPError{
				Status:     resp.StatusCode,
				URL:        endpoint,
				RetryAfter: retryDelay,
			})
		}

		// Prefer Retry-After when given (capped); else exponential backoff.
		// The earlier implementation slept twice on 429 — once for Retry-After
		// and again for the index-based delays slice. We pick exactly one.
		var delay time.Duration
		switch {
		case retryDelay > 0 && retryDelay <= policy.maxRetryAfterDelay:
			delay = retryDelay
		case retryDelay > policy.maxRetryAfterDelay:
			// Bail out instead of blocking the CLI for minutes.
			return "", wrapHTTPStatus(&HTTPError{
				Status:     resp.StatusCode,
				URL:        endpoint,
				RetryAfter: retryDelay,
			})
		default:
			delay = backoffFor(attempt, policy.defaultBackoff)
		}
		c.logRetry(RetryEvent{
			URL:         endpoint,
			Status:      resp.StatusCode,
			Attempt:     attempt + 1,
			MaxAttempts: policy.maxAttempts,
			Delay:       delay,
			RetryAfter:  retryDelay > 0 && retryDelay <= policy.maxRetryAfterDelay,
		})
		time.Sleep(delay)
	}
	if lastErr != nil {
		return "", wrapNetwork(lastErr, endpoint)
	}
	return "", &Error{Code: CodeUnknown, Message: fmt.Sprintf("request failed for %s", endpoint)}
}

func (c *Client) logRetry(event RetryEvent) {
	if c.RetryLogger != nil {
		c.RetryLogger(event)
	}
}

func backoffFor(attempt int, base time.Duration) time.Duration {
	d := base
	for i := 0; i < attempt; i++ {
		d *= 2
	}
	return d
}

func readAllAndClose(rc io.ReadCloser) ([]byte, error) {
	defer rc.Close()
	return io.ReadAll(rc)
}

func (c *Client) waitForInterval() {
	if c.MinInterval <= 0 {
		return
	}
	c.rateMu.Lock()
	defer c.rateMu.Unlock()
	wait := c.MinInterval - time.Since(c.lastReq)
	if wait > 0 {
		time.Sleep(wait)
	}
	c.lastReq = time.Now()
}

// retryAfterDuration parses an HTTP Retry-After header (seconds or HTTP-date).
// Returns zero when absent or unparseable.
func retryAfterDuration(value string) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		delay := time.Until(when)
		if delay > 0 {
			return delay
		}
	}
	return 0
}

// retryAfter is kept as a thin alias to preserve the existing test name.
func retryAfter(value string) time.Duration { return retryAfterDuration(value) }

// silence linter for currently unused errors helper. Will be used by tests.
var _ = errors.As
