package cmd

import (
	"context"
	"errors"
	"os"

	"steam-cli/internal/i18n"
	"steam-cli/internal/steam"
)

// classifyError maps any error returned by a command to the JSON envelope's
// error.type / hint pair. It uses errors.As against the typed errors from
// internal/steam rather than substring matching on Error().
func classifyError(err error) jsonError {
	out := jsonError{Type: string(steam.CodeUnknown), Message: err.Error()}

	// Honor an explicit HintKey when present (set by package steam).
	if hintKey := steam.HintKeyOf(err); hintKey != "" {
		out.Hint = i18n.T(hintKey)
	}

	// Walk the error chain for a typed Code, then fall back to context errors.
	code := steam.CodeOf(err)
	if code == steam.CodeUnknown {
		if errors.Is(err, context.DeadlineExceeded) {
			code = steam.CodeNetworkTimeout
		}
	}
	out.Type = string(code)
	if out.Hint == "" {
		out.Hint = hintFor(code)
	}
	return out
}

// hintFor returns the localized default hint for a given Code.
// HintKey on a typed error always wins over this default.
func hintFor(code steam.Code) string {
	switch code {
	case steam.CodeRateLimited:
		return i18n.T("hint.rate_limited")
	case steam.CodeAccessDenied:
		return i18n.T("hint.access_denied")
	case steam.CodePrivacyRestricted:
		return i18n.T("hint.privacy_restricted")
	case steam.CodeNotFound:
		return i18n.T("hint.not_found")
	case steam.CodeInvalidInput:
		return i18n.T("hint.invalid_appid")
	case steam.CodeNetworkTimeout:
		return i18n.T("hint.network_timeout")
	case steam.CodeSourceChanged:
		return i18n.T("hint.source_changed")
	}
	return ""
}

func errorCommandPath() string {
	if currentCommand != "" {
		return currentCommand
	}
	if cmd, _, err := rootCmd.Find(os.Args[1:]); err == nil && cmd != nil {
		return cmd.CommandPath()
	}
	return rootCmd.CommandPath()
}
