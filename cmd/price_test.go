package cmd

import (
	"strings"
	"testing"
	"time"

	"steam-cli/internal/steam"
)

func TestRegionalClientSharesBaseConfiguration(t *testing.T) {
	base := steam.NewClient("US", "english", time.Second)
	base.Cache = steam.NewCache()
	base.MinInterval = 125 * time.Millisecond

	region := regionalClient(base, "cn")

	if region == base {
		t.Fatal("regionalClient should build a sibling client, not reuse the base pointer")
	}
	if region.CC != "CN" {
		t.Fatalf("region.CC = %q, want CN", region.CC)
	}
	if region.Lang != base.Lang {
		t.Fatalf("region.Lang = %q, want %q", region.Lang, base.Lang)
	}
	if region.HTTPClient != base.HTTPClient {
		t.Fatal("regional client should share HTTPClient")
	}
	if region.Cache != base.Cache {
		t.Fatal("regional client should share Cache")
	}
	if region.Endpoints != base.Endpoints {
		t.Fatalf("region.Endpoints = %#v, want %#v", region.Endpoints, base.Endpoints)
	}
	if region.MinInterval != base.MinInterval {
		t.Fatalf("region.MinInterval = %s, want %s", region.MinInterval, base.MinInterval)
	}
}

func TestReviewSummaryTextHandlesNil(t *testing.T) {
	if got := reviewScoreText(nil); got != "-" {
		t.Fatalf("reviewScoreText(nil) = %q, want -", got)
	}
	if got := reviewCountText(nil, func(r *steam.ReviewSummary) int { return r.TotalReviews }); got != "-" {
		t.Fatalf("reviewCountText(nil) = %q, want -", got)
	}
}

func TestReviewSummaryTextFormatsValues(t *testing.T) {
	summary := &steam.ReviewSummary{
		ReviewScoreDesc: "Very Positive",
		TotalReviews:    42,
	}
	if got := reviewScoreText(summary); got != "Very Positive" {
		t.Fatalf("reviewScoreText() = %q", got)
	}
	if got := reviewCountText(summary, func(r *steam.ReviewSummary) int { return r.TotalReviews }); got != "42" {
		t.Fatalf("reviewCountText() = %q", got)
	}
}

// formatDiscountEnd should put local time first, then UTC and PT in
// parentheses. The chosen instant is an arbitrary 2026-05-26 01:00 UTC+08:00
// = 2026-05-25 17:00 UTC = 2026-05-25 10:00 PDT.
func TestFormatDiscountEndShanghaiShowsAllZones(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Skipf("Asia/Shanghai not available: %v", err)
	}
	unix := time.Date(2026, 5, 25, 17, 0, 0, 0, time.UTC).Unix()
	got := formatDiscountEndIn(unix, shanghai)

	if !strings.HasPrefix(got, "2026-05-26 01:00 UTC+08:00 ") {
		t.Errorf("expected local time first with UTC offset; got %q", got)
	}
	if !strings.Contains(got, "UTC 2026-05-25 17:00") {
		t.Errorf("missing UTC line; got %q", got)
	}
	if !strings.Contains(got, "PT 2026-05-25 10:00") {
		t.Errorf("missing PT line; got %q", got)
	}
	// Local should come before the parenthesized list.
	if i := strings.Index(got, "("); i < 0 || i < strings.Index(got, "UTC+08:00") {
		t.Errorf("local time should precede the parens; got %q", got)
	}
}

func TestFormatDiscountEndUTCSkipsUTCLine(t *testing.T) {
	unix := time.Date(2026, 5, 25, 17, 0, 0, 0, time.UTC).Unix()
	got := formatDiscountEndIn(unix, time.UTC)

	if !strings.Contains(got, "UTC+00:00") {
		t.Errorf("UTC location should render as UTC+00:00; got %q", got)
	}
	if strings.Contains(got, "UTC 2026") {
		t.Errorf("UTC line is redundant when local IS UTC; got %q", got)
	}
	if !strings.Contains(got, "PT ") {
		t.Errorf("PT should still be shown for UTC users; got %q", got)
	}
}

func TestFormatDiscountEndPTSkipsPTLine(t *testing.T) {
	pt, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skipf("America/Los_Angeles not available: %v", err)
	}
	unix := time.Date(2026, 5, 25, 17, 0, 0, 0, time.UTC).Unix()
	got := formatDiscountEndIn(unix, pt)

	if strings.Contains(got, "PT 2026") {
		t.Errorf("PT line is redundant when local IS PT; got %q", got)
	}
	if !strings.Contains(got, "UTC 2026") {
		t.Errorf("UTC line should still be shown for PT users; got %q", got)
	}
}

// localTimeFormat must include an explicit UTC offset, not an abbreviation.
// The ambiguous "CST" abbreviation (China Standard Time vs Central Standard
// Time) is the reason we moved away from "MST" formatting.
func TestLocalTimeFormatUsesUTCOffset(t *testing.T) {
	if !strings.Contains(localTimeFormat, "UTC-07:00") {
		t.Fatalf("localTimeFormat should use an explicit UTC offset; got %q", localTimeFormat)
	}
	if strings.Contains(localTimeFormat, "MST") {
		t.Fatalf("localTimeFormat should not use MST abbreviation; got %q", localTimeFormat)
	}
}
