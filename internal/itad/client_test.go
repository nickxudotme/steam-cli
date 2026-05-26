package itad

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"steam-cli/internal/steam"
)

func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c := NewClient("test-key", 5*time.Second)
	c.BaseURL = srv.URL
	return c
}

func TestLookupByAppID(t *testing.T) {
	var query url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		if got := r.Header.Get("ITAD-API-Key"); got != "test-key" {
			t.Fatalf("ITAD-API-Key = %q, want test-key", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"found": true,
			"game": map[string]any{
				"id":    "gid-1",
				"title": "Subnautica",
				"type":  "game",
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	game, err := client.LookupByAppID(264710)
	if err != nil {
		t.Fatalf("LookupByAppID returned error: %v", err)
	}
	if query.Get("appid") != "264710" {
		t.Fatalf("appid query = %q, want 264710", query.Get("appid"))
	}
	if game.ID != "gid-1" || game.AppID != 264710 {
		t.Fatalf("game = %#v", game)
	}
}

func TestSummaryByAppIDLoadsOverviewHistoryAndSteamLow(t *testing.T) {
	var called []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = append(called, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/games/lookup/v1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"found": true,
				"game": map[string]any{
					"id":    "gid-1",
					"title": "Subnautica",
					"type":  "game",
				},
			})
		case "/games/overview/v2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"prices": []map[string]any{{
					"id": "gid-1",
					"current": map[string]any{
						"shop":      map[string]any{"id": 6, "name": "Fanatical"},
						"price":     map[string]any{"amount": 7.49, "amountInt": 749, "currency": "USD"},
						"regular":   map[string]any{"amount": 29.99, "amountInt": 2999, "currency": "USD"},
						"cut":       75,
						"timestamp": "2026-05-27T12:00:00Z",
						"url":       "https://itad.link/deal",
					},
					"lowest": map[string]any{
						"shop":      map[string]any{"id": 6, "name": "Fanatical"},
						"price":     map[string]any{"amount": 5.99, "amountInt": 599, "currency": "USD"},
						"regular":   map[string]any{"amount": 29.99, "amountInt": 2999, "currency": "USD"},
						"cut":       80,
						"timestamp": "2025-12-01T12:00:00Z",
					},
					"bundled": 1,
					"urls":    map[string]any{"game": "https://isthereanydeal.com/game/subnautica"},
				}},
				"bundles": []map[string]any{{
					"id":    10,
					"title": "Ocean Bundle",
					"url":   "https://bundle.example",
					"tiers": []map[string]any{},
				}},
			})
		case "/games/historylow/v1":
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id": "gid-1",
				"historyLow": map[string]any{
					"all": map[string]any{"amount": 5.99, "amountInt": 599, "currency": "USD"},
				},
				"deals": []map[string]any{},
			}})
		case "/games/storelow/v2":
			if !strings.Contains(r.URL.RawQuery, "shops=61") {
				t.Fatalf("storelow query = %s, want shops=61", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id": "gid-1",
				"low": map[string]any{
					"shop":      map[string]any{"id": 61, "name": "Steam"},
					"price":     map[string]any{"amount": 7.49, "amountInt": 749, "currency": "USD"},
					"regular":   map[string]any{"amount": 29.99, "amountInt": 2999, "currency": "USD"},
					"cut":       75,
					"timestamp": "2026-01-01T12:00:00Z",
				},
			}})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	summary, err := client.SummaryByAppID(264710, "US")
	if err != nil {
		t.Fatalf("SummaryByAppID returned error: %v", err)
	}
	if summary.Game.ID != "gid-1" {
		t.Fatalf("summary.Game.ID = %q, want gid-1", summary.Game.ID)
	}
	if summary.Overview == nil || summary.Overview.Current == nil {
		t.Fatalf("summary.Overview = %#v", summary.Overview)
	}
	if summary.HistoryLow == nil || summary.HistoryLow.HistoryLow.All == nil {
		t.Fatalf("summary.HistoryLow = %#v", summary.HistoryLow)
	}
	if summary.SteamLow == nil || summary.SteamLow.Low == nil {
		t.Fatalf("summary.SteamLow = %#v", summary.SteamLow)
	}
	if len(summary.Bundles) != 1 {
		t.Fatalf("len(summary.Bundles) = %d, want 1", len(summary.Bundles))
	}
	if len(called) != 4 {
		t.Fatalf("called %d endpoints, want 4", len(called))
	}
}

func TestShops(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": 61, "title": "Steam", "deals": 10, "games": 20},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	shops, err := client.Shops()
	if err != nil {
		t.Fatalf("Shops returned error: %v", err)
	}
	if len(shops) != 1 || shops[0].ID != 61 || shops[0].Title != "Steam" {
		t.Fatalf("shops = %#v", shops)
	}
}

func TestMissingKeyReturnsTypedError(t *testing.T) {
	client := NewClient("", time.Second)
	_, err := client.Shops()
	if err == nil {
		t.Fatal("expected error")
	}
	if steam.CodeOf(err) != steam.CodeAccessDenied {
		t.Fatalf("steam.CodeOf(err) = %s, want access_denied", steam.CodeOf(err))
	}
}
