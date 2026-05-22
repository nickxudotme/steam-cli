package cmd

import (
	"strings"
	"testing"

	"steam-cli/internal/steam"
)

func TestStoreReviewText(t *testing.T) {
	got := storeReviewText(&steam.StoreReviews{SummaryFiltered: &steam.StoreReviewSummary{
		ReviewCount:      1000,
		PercentPositive:  97,
		ReviewScoreLabel: "Overwhelmingly Positive",
	}})
	if !strings.Contains(got, "Overwhelmingly Positive") || !strings.Contains(got, "97%") {
		t.Fatalf("storeReviewText() = %q", got)
	}
}

func TestStorePlatformsText(t *testing.T) {
	got := storePlatformsText(&steam.StorePlatforms{
		Windows: true,
		Mac:     true,
		VRSupport: &steam.VRSupport{
			VRHMD: true,
		},
	})
	if got != "windows, mac, vr" {
		t.Fatalf("storePlatformsText() = %q", got)
	}
}

func TestStoreLanguageSummary(t *testing.T) {
	got := storeLanguageSummary([]steam.StoreLanguage{
		{Supported: true, FullAudio: true, Subtitles: true},
		{Supported: true, Subtitles: true},
	})
	if got != "2 supported, 1 full audio, 2 subtitles" {
		t.Fatalf("storeLanguageSummary() = %q", got)
	}
}

func TestPurchaseOptionID(t *testing.T) {
	if got := purchaseOptionID(steam.PurchaseOption{PackageID: 123}); got != "sub 123" {
		t.Fatalf("package purchaseOptionID() = %q", got)
	}
	if got := purchaseOptionID(steam.PurchaseOption{BundleID: 456}); got != "bundle 456" {
		t.Fatalf("bundle purchaseOptionID() = %q", got)
	}
}
