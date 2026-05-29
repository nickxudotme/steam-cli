package cmd

import (
	"testing"
	"time"

	"steam-cli/internal/i18n"
	"steam-cli/internal/itad"
)

func TestParseShopIDsDeduplicatesAndNormalizes(t *testing.T) {
	got, err := parseShopIDs("61, 35,61")
	if err != nil {
		t.Fatalf("parseShopIDs returned error: %v", err)
	}
	want := []int{61, 35}
	if len(got) != len(want) {
		t.Fatalf("parseShopIDs() = %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseShopIDs()[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestParseShopIDsRejectsNonNumeric(t *testing.T) {
	if _, err := parseShopIDs("steam"); err == nil {
		t.Fatal("expected error")
	}
}

func TestActiveITADBundlesSkipsExpired(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	future := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	bundles := []itad.Bundle{
		{ID: 1, Expiry: &past},
		{ID: 2, Expiry: &future},
		{ID: 3, Expiry: nil},
	}
	got := activeITADBundles(bundles)
	if len(got) != 2 {
		t.Fatalf("len(activeITADBundles()) = %d, want 2", len(got))
	}
	if got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("activeITADBundles() = %#v", got)
	}
}

func TestCommandSourcesIncludeAdvancedPricingWhenEnabled(t *testing.T) {
	origPrice := priceOpts.enhanced
	origApp := appOpts.enhanced
	priceOpts.enhanced = true
	appOpts.enhanced = true
	defer func() {
		priceOpts.enhanced = origPrice
		appOpts.enhanced = origApp
	}()

	priceSources := commandSources("steam-cli price")
	appSources := commandSources("steam-cli app")
	if len(priceSources) < 4 {
		t.Fatalf("priceSources = %#v", priceSources)
	}
	if len(appSources) < 6 {
		t.Fatalf("appSources = %#v", appSources)
	}
}

func TestBuildSaleWindowsDerivesDiscountRanges(t *testing.T) {
	entries := []itad.HistoryEntry{
		{
			Timestamp: "2026-05-25T00:00:00Z",
			Shop:      itad.ShopRef{Name: "Steam"},
			Deal: struct {
				Price   itad.Money `json:"price"`
				Regular itad.Money `json:"regular"`
				Cut     int        `json:"cut"`
			}{
				Price:   itad.Money{Amount: 29.99, Currency: "USD"},
				Regular: itad.Money{Amount: 29.99, Currency: "USD"},
				Cut:     0,
			},
		},
		{
			Timestamp: "2026-05-11T00:00:00Z",
			Shop:      itad.ShopRef{Name: "Steam"},
			Deal: struct {
				Price   itad.Money `json:"price"`
				Regular itad.Money `json:"regular"`
				Cut     int        `json:"cut"`
			}{
				Price:   itad.Money{Amount: 7.49, Currency: "USD"},
				Regular: itad.Money{Amount: 29.99, Currency: "USD"},
				Cut:     75,
			},
		},
		{
			Timestamp: "2026-04-26T00:00:00Z",
			Shop:      itad.ShopRef{Name: "Steam"},
			Deal: struct {
				Price   itad.Money `json:"price"`
				Regular itad.Money `json:"regular"`
				Cut     int        `json:"cut"`
			}{
				Price:   itad.Money{Amount: 29.99, Currency: "USD"},
				Regular: itad.Money{Amount: 29.99, Currency: "USD"},
				Cut:     0,
			},
		},
	}
	got := buildSaleWindows(entries)
	if len(got) != 1 {
		t.Fatalf("len(buildSaleWindows()) = %d, want 1", len(got))
	}
	if got[0].Start != "2026-05-11" || got[0].End != "2026-05-25" || got[0].Discount != "-75%" {
		t.Fatalf("buildSaleWindows()[0] = %#v", got[0])
	}
	if got[0].StartAt != "2026-05-11T00:00:00Z" || got[0].EndAt != "2026-05-25T00:00:00Z" {
		t.Fatalf("buildSaleWindows()[0] = %#v", got[0])
	}
	if got[0].StartUnix != 1778457600 || got[0].EndUnix != 1779667200 {
		t.Fatalf("buildSaleWindows()[0] = %#v", got[0])
	}
}

func TestBuildSaleWindowsMarksMostRecentDiscountActive(t *testing.T) {
	entries := []itad.HistoryEntry{
		{
			Timestamp: "2026-05-25T00:00:00Z",
			Shop:      itad.ShopRef{Name: "Steam"},
			Deal: struct {
				Price   itad.Money `json:"price"`
				Regular itad.Money `json:"regular"`
				Cut     int        `json:"cut"`
			}{
				Price:   itad.Money{Amount: 7.49, Currency: "USD"},
				Regular: itad.Money{Amount: 29.99, Currency: "USD"},
				Cut:     75,
			},
		},
	}
	got := buildSaleWindows(entries)
	if len(got) != 1 {
		t.Fatalf("len(buildSaleWindows()) = %d, want 1", len(got))
	}
	if got[0].Status != i18n.T("history.status_active") || got[0].End != "" {
		t.Fatalf("buildSaleWindows()[0] = %#v", got[0])
	}
	if got[0].StartAt != "2026-05-25T00:00:00Z" || got[0].StartUnix != 1779667200 {
		t.Fatalf("buildSaleWindows()[0] = %#v", got[0])
	}
	if got[0].EndAt != "" || got[0].EndUnix != 0 {
		t.Fatalf("buildSaleWindows()[0] = %#v", got[0])
	}
}
