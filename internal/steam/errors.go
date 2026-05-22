package steam

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Code is a stable, machine-readable classification used by the CLI to render
// localized hints and to populate the JSON envelope's error.type field.
type Code string

const (
	CodeRateLimited       Code = "rate_limited"
	CodeAccessDenied      Code = "access_denied"
	CodePrivacyRestricted Code = "privacy_restricted"
	CodeNotFound          Code = "not_found"
	CodeInvalidInput      Code = "invalid_input"
	CodeNetworkTimeout    Code = "network_timeout"
	CodeSourceChanged     Code = "source_changed"
	CodeUnknown           Code = "unknown"
)

// Error is the typed error returned by everything in this package. Callers
// should use errors.As(&Error{}) to classify failures rather than matching
// substrings of Error.Error().
type Error struct {
	Code    Code
	Message string
	// HintKey is an optional i18n key the CLI will render as the user-facing
	// hint. When empty, the CLI falls back to a hint chosen from Code.
	HintKey string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

// HTTPError represents a non-2xx status from a Steam endpoint. The retry loop
// returns this on its final attempt; it is also wrapped by typed Errors when
// HTTP status implies a higher-level Code (e.g. 429 -> CodeRateLimited).
type HTTPError struct {
	Status     int
	URL        string
	RetryAfter time.Duration
}

func (e *HTTPError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("HTTP %d from %s: retry after %s", e.Status, e.URL, e.RetryAfter)
	}
	return fmt.Sprintf("HTTP %d from %s", e.Status, e.URL)
}

// HTTPErrorFromAny extracts a *HTTPError from any wrapped error chain.
func HTTPErrorFromAny(err error) (*HTTPError, bool) {
	var he *HTTPError
	if errors.As(err, &he) {
		return he, true
	}
	return nil, false
}

// CodeOf returns the Code embedded in err, walking errors.As / errors.Is.
// HTTP statuses without a wrapping *Error get mapped here.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}
	var typed *Error
	if errors.As(err, &typed) {
		return typed.Code
	}
	if he, ok := HTTPErrorFromAny(err); ok {
		switch {
		case he.Status == 429:
			return CodeRateLimited
		case he.Status == 401, he.Status == 403:
			return CodeAccessDenied
		case he.Status == 404:
			return CodeNotFound
		}
	}
	if isTimeout(err) {
		return CodeNetworkTimeout
	}
	return CodeUnknown
}

// HintKeyOf returns the explicit HintKey on a typed error if set, else "".
// The CLI falls back to a code-based default when this is empty.
func HintKeyOf(err error) string {
	var typed *Error
	if errors.As(err, &typed) {
		return typed.HintKey
	}
	return ""
}

// Constructors used internally by this package.

func newInvalidInput(format string, args ...any) *Error {
	return &Error{Code: CodeInvalidInput, Message: fmt.Sprintf(format, args...)}
}

func newInvalidProfileInput(format string, args ...any) *Error {
	return &Error{
		Code:    CodeInvalidInput,
		Message: fmt.Sprintf(format, args...),
		HintKey: "hint.invalid_profile_input",
	}
}

func newNotFound(format string, args ...any) *Error {
	return &Error{Code: CodeNotFound, Message: fmt.Sprintf(format, args...)}
}

func newPrivacyRestricted(format string, args ...any) *Error {
	return &Error{Code: CodePrivacyRestricted, Message: fmt.Sprintf(format, args...)}
}

func newSourceChanged(cause error, endpoint string) *Error {
	return &Error{
		Code:    CodeSourceChanged,
		Message: fmt.Sprintf("invalid response from %s", endpoint),
		Cause:   cause,
	}
}

// wrapHTTPStatus turns an *HTTPError into a typed *Error using the canonical
// Code mapping. The HTTPError stays in the chain so HTTPErrorFromAny works.
func wrapHTTPStatus(he *HTTPError) *Error {
	switch {
	case he.Status == 429:
		return &Error{Code: CodeRateLimited, Message: he.Error(), Cause: he}
	case he.Status == 401, he.Status == 403:
		return &Error{Code: CodeAccessDenied, Message: he.Error(), Cause: he}
	case he.Status == 404:
		return &Error{Code: CodeNotFound, Message: he.Error(), Cause: he}
	default:
		return &Error{Code: CodeUnknown, Message: he.Error(), Cause: he}
	}
}

func wrapNetwork(err error, endpoint string) error {
	if err == nil {
		return nil
	}
	if isTimeout(err) {
		return &Error{
			Code:    CodeNetworkTimeout,
			Message: fmt.Sprintf("network timeout for %s", endpoint),
			Cause:   err,
		}
	}
	return &Error{
		Code:    CodeUnknown,
		Message: fmt.Sprintf("request failed for %s", endpoint),
		Cause:   err,
	}
}

func isTimeout(err error) bool {
	type timeoutError interface{ Timeout() bool }
	var t timeoutError
	if errors.As(err, &t) && t.Timeout() {
		return true
	}
	var ue *url.Error
	if errors.As(err, &ue) && ue.Timeout() {
		return true
	}
	return false
}
