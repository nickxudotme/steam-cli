package cmd

import (
	"errors"
	"testing"

	"steam-cli/internal/steam"
)

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
