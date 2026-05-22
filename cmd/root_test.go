package cmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	"steam-cli/internal/i18n"
	"steam-cli/internal/steam"
)

// i18nReset clears the cached system-locale detection so tests that change
// LC_* env vars see a fresh result.
func i18nReset() { i18n.ResetDetect() }

func TestClassifyErrorInvalidAppID(t *testing.T) {
	got := classifyError(&steam.Error{
		Code:    steam.CodeInvalidInput,
		Message: "invalid appid \"abc\"",
		HintKey: "hint.invalid_appid",
	})
	if got.Type != "invalid_input" || got.Hint == "" {
		t.Fatalf("classifyError() = %#v", got)
	}
}

func TestClassifyErrorRateLimited(t *testing.T) {
	err := &steam.HTTPError{Status: 429, URL: "https://example.com"}
	wrapped := steam.CodeOf(err)
	if wrapped != steam.CodeRateLimited {
		t.Fatalf("CodeOf(429) = %s", wrapped)
	}
	got := classifyError(err)
	if got.Type != "rate_limited" || got.Hint == "" {
		t.Fatalf("classifyError(429) = %#v", got)
	}
}

func TestClassifyErrorWrappedHTTPError(t *testing.T) {
	wrapped := errors.Join(errors.New("ctx"), &steam.Error{
		Code:    steam.CodeAccessDenied,
		Message: "denied",
		Cause:   &steam.HTTPError{Status: 403, URL: "u"},
	})
	got := classifyError(wrapped)
	if got.Type != "access_denied" || got.Hint == "" {
		t.Fatalf("classifyError(wrapped) = %#v", got)
	}
}

func TestClassifyErrorUnknownStaysUnknown(t *testing.T) {
	got := classifyError(errors.New("nothing typed here"))
	if got.Type != "unknown" {
		t.Fatalf("expected unknown, got %s", got.Type)
	}
}

func TestSplitCodesDeduplicatesAndNormalizes(t *testing.T) {
	got := splitCodes("cn, US,cn,,jp")
	want := []string{"CN", "US", "JP"}
	if len(got) != len(want) {
		t.Fatalf("splitCodes() = %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitCodes()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRetryNoticeFormatsRateLimit(t *testing.T) {
	got := retryNotice(steam.RetryEvent{
		URL:         "https://api.steampowered.com/IWishlistService/GetWishlist/v1/",
		Status:      429,
		Attempt:     1,
		MaxAttempts: 3,
		Delay:       5 * time.Second,
	})
	if !strings.Contains(got, "rate limited") || !strings.Contains(got, "retrying in 5s") || !strings.Contains(got, "attempt 2/3") {
		t.Fatalf("retryNotice() = %q", got)
	}
}

func TestValidateEnumFlagRejectsUnknownValue(t *testing.T) {
	err := validateEnumFlag("filter", "nonsense", "specials", "topsellers")
	if err == nil {
		t.Fatal("expected invalid enum error")
	}
	if steam.CodeOf(err) != steam.CodeInvalidInput {
		t.Fatalf("expected CodeInvalidInput, got %s", steam.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "specials, topsellers") {
		t.Fatalf("expected allowed values in error, got %q", err.Error())
	}
}

func TestExactArgsWithExample(t *testing.T) {
	cmd := rootCmd
	err := exactArgsWithExample(1, "steam-cli search TERM", "steam-cli search portal")(cmd, nil)
	if err == nil {
		t.Fatal("expected arg error")
	}
	if steam.CodeOf(err) != steam.CodeInvalidInput {
		t.Fatalf("expected CodeInvalidInput, got %s", steam.CodeOf(err))
	}
	if !strings.Contains(err.Error(), "Usage:") || !strings.Contains(err.Error(), "Example:") {
		t.Fatalf("expected usage and example in error, got %q", err.Error())
	}
}

func TestParseAppIDValid(t *testing.T) {
	got, err := parseAppID(" 264710 ")
	if err != nil {
		t.Fatalf("parseAppID returned error: %v", err)
	}
	if got != 264710 {
		t.Fatalf("parseAppID() = %d", got)
	}
}

func TestParseAppIDRejectsNonNumeric(t *testing.T) {
	_, err := parseAppID("not-an-app")
	if err == nil {
		t.Fatal("expected error")
	}
	if steam.CodeOf(err) != steam.CodeInvalidInput {
		t.Fatalf("expected CodeInvalidInput, got %s", steam.CodeOf(err))
	}
}

func TestParseAppIDRejectsZero(t *testing.T) {
	if _, err := parseAppID("0"); err == nil {
		t.Fatal("expected error for 0")
	}
}

// TestNormalizeOptionsAuto validates that "auto" sentinel values for --cc and
// --lang resolve via system locale detection. Uses LC_ALL to force a known
// locale; ResetDetect avoids cache leakage from earlier tests.
func TestNormalizeOptionsAuto(t *testing.T) {
	i18nReset()
	t.Setenv("LC_ALL", "ja_JP.UTF-8")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "")

	opts.cc = "auto"
	opts.lang = "auto"
	opts.uiLang = "en"
	defer func() {
		opts.cc, opts.lang, opts.uiLang = "auto", "auto", "auto"
		i18nReset()
	}()

	normalizeOptions()

	if opts.cc != "JP" {
		t.Fatalf("opts.cc = %q, want JP", opts.cc)
	}
	if opts.lang != "japanese" {
		t.Fatalf("opts.lang = %q, want japanese", opts.lang)
	}
}

// TestNormalizeOptionsExplicitOverride confirms that explicit values are not
// stomped by auto-detection, even when they look unusual.
func TestNormalizeOptionsExplicitOverride(t *testing.T) {
	i18nReset()
	t.Setenv("LC_ALL", "ja_JP.UTF-8")

	opts.cc = "DE"
	opts.lang = "german"
	opts.uiLang = "en"
	defer func() {
		opts.cc, opts.lang, opts.uiLang = "auto", "auto", "auto"
		i18nReset()
	}()

	normalizeOptions()

	if opts.cc != "DE" {
		t.Fatalf("opts.cc = %q, want DE (no auto override)", opts.cc)
	}
	if opts.lang != "german" {
		t.Fatalf("opts.lang = %q, want german (no auto override)", opts.lang)
	}
}
