package i18n

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// SystemLocale captures the parsed components of an OS-level locale
// identifier such as "zh_CN.UTF-8", "en-US", or "zh-Hant-TW".
//
// Language is a lowercase ISO 639-1 code (e.g. "zh", "en", "ja").
// Region is an uppercase ISO 3166-1 alpha-2 code (e.g. "CN", "US", "JP").
// Script is the writing system when present (e.g. "Hans", "Hant").
type SystemLocale struct {
	Language string
	Region   string
	Script   string
}

// DetectSystemLocale resolves the user's locale from, in order:
//  1. LC_ALL / LC_MESSAGES / LC_MONETARY / LANG environment variables
//  2. OS-specific sources (AppleLocale on macOS, Get-Culture on Windows,
//     /etc/locale.conf or /etc/default/locale on Linux)
//  3. Conservative fallback "en" / "US"
//
// Result is sync.Once-cached. Tests touching env vars must call ResetDetect
// first so the cache doesn't leak.
func DetectSystemLocale() SystemLocale {
	systemLocaleOnce.Do(func() {
		if loc, ok := computeSystemLocaleFromEnv(os.Getenv); ok {
			systemLocale = loc
			return
		}
		if loc, ok := detectSystemLocaleFromOS(); ok {
			systemLocale = loc
			return
		}
		systemLocale = SystemLocale{Language: "en", Region: "US"}
	})
	return systemLocale
}

var (
	systemLocaleOnce sync.Once
	systemLocale     SystemLocale
)

// ResetDetect clears the cached system-locale detection. Tests that mutate
// env vars must call this before invoking auto-detect.
func ResetDetect() {
	systemLocaleOnce = sync.Once{}
	systemLocale = SystemLocale{}
}

func computeSystemLocaleFromEnv(getenv func(string) string) (SystemLocale, bool) {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LC_MONETARY", "LANG"} {
		if loc, ok := parseLocale(getenv(key)); ok {
			return loc, true
		}
	}
	return SystemLocale{}, false
}

// parseLocale parses common locale identifier shapes:
//   - "xx" / "xx_YY" / "xx-YY"                        (POSIX, Windows)
//   - "xx_YY.encoding" / "xx_YY@modifier"             (POSIX with extras)
//   - "xx-Script-YY" / "xx_Script_YY" (e.g. zh-Hant-TW, BCP-47)
//
// Returns ok=false for empty input or neutral locales like "C", "POSIX",
// "C.UTF-8".
func parseLocale(value string) (SystemLocale, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return SystemLocale{}, false
	}
	if i := strings.IndexAny(value, ".@"); i >= 0 {
		value = value[:i]
	}
	if value == "" || isNeutralLocale(strings.ToLower(value)) {
		return SystemLocale{}, false
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-'
	})
	if len(parts) == 0 {
		return SystemLocale{}, false
	}
	loc := SystemLocale{Language: strings.ToLower(parts[0])}
	for _, p := range parts[1:] {
		switch len(p) {
		case 4:
			// Script subtag, e.g. "Hans" / "Hant".
			loc.Script = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		case 2, 3:
			loc.Region = strings.ToUpper(p)
		}
	}
	return loc, true
}

func detectSystemLocaleFromOS() (SystemLocale, bool) {
	switch runtime.GOOS {
	case "darwin":
		// AppleLocale gives a single "lang_REGION" identifier when set.
		if loc, ok := commandLocale(1500*time.Millisecond, "defaults", "read", "-g", "AppleLocale"); ok {
			return loc, true
		}
		// AppleLanguages returns an array; first entry typically like
		// "zh-Hans-CN" or "en-US".
		return commandLocale(1500*time.Millisecond, "defaults", "read", "-g", "AppleLanguages")
	case "windows":
		if loc, ok := commandLocale(2500*time.Millisecond, "powershell", "-NoProfile", "-Command", "(Get-Culture).Name"); ok {
			return loc, true
		}
		return commandLocale(2500*time.Millisecond, "powershell", "-NoProfile", "-Command", "(Get-WinSystemLocale).Name")
	case "linux":
		for _, path := range []string{"/etc/locale.conf", "/etc/default/locale"} {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if loc, ok := parseLocaleConfFile(string(data)); ok {
				return loc, true
			}
		}
	}
	return SystemLocale{}, false
}

func parseLocaleConfFile(text string) (SystemLocale, bool) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "LANG=") && !strings.HasPrefix(line, "LC_ALL=") {
			continue
		}
		_, value, _ := strings.Cut(line, "=")
		value = strings.Trim(value, `"' `)
		if loc, ok := parseLocale(value); ok {
			return loc, true
		}
	}
	return SystemLocale{}, false
}

func commandLocale(timeout time.Duration, name string, args ...string) (SystemLocale, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return SystemLocale{}, false
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// Strip array/quote decorations from `defaults` output.
		line = strings.Trim(line, `()"',`)
		if loc, ok := parseLocale(line); ok {
			return loc, true
		}
	}
	return SystemLocale{}, false
}

// SteamLangFor maps a SystemLocale to a Steam --lang code. Returns the empty
// string when no Steam language fits.
func SteamLangFor(loc SystemLocale) string {
	switch loc.Language {
	case "zh":
		if loc.Script == "Hant" || loc.Region == "TW" || loc.Region == "HK" || loc.Region == "MO" {
			return "tchinese"
		}
		return "schinese"
	case "en":
		return "english"
	case "ja":
		return "japanese"
	case "ko":
		return "koreana"
	case "th":
		return "thai"
	case "bg":
		return "bulgarian"
	case "cs":
		return "czech"
	case "da":
		return "danish"
	case "de":
		return "german"
	case "es":
		// Latin American Spanish gets a dedicated Steam lang code.
		switch loc.Region {
		case "AR", "BO", "CL", "CO", "CR", "CU", "DO", "EC", "GT", "HN",
			"MX", "NI", "PA", "PE", "PR", "PY", "SV", "UY", "VE":
			return "latam"
		}
		return "spanish"
	case "el":
		return "greek"
	case "fr":
		return "french"
	case "it":
		return "italian"
	case "id":
		return "indonesian"
	case "hu":
		return "hungarian"
	case "nl":
		return "dutch"
	case "no", "nb", "nn":
		return "norwegian"
	case "pl":
		return "polish"
	case "pt":
		if loc.Region == "BR" {
			return "brazilian"
		}
		return "portuguese"
	case "ro":
		return "romanian"
	case "ru":
		return "russian"
	case "fi":
		return "finnish"
	case "sv":
		return "swedish"
	case "tr":
		return "turkish"
	case "vi":
		return "vietnamese"
	case "uk":
		return "ukrainian"
	case "ar":
		return "arabic"
	}
	return ""
}

// SteamCCFor maps a SystemLocale to a Steam --cc (country/region) code. If
// the locale has an explicit region, that wins; otherwise we fall back to a
// language-implied default. Returns the empty string when no mapping fits.
func SteamCCFor(loc SystemLocale) string {
	if len(loc.Region) == 2 {
		return strings.ToUpper(loc.Region)
	}
	switch loc.Language {
	case "zh":
		if loc.Script == "Hant" {
			return "TW"
		}
		return "CN"
	case "en":
		return "US"
	case "ja":
		return "JP"
	case "ko":
		return "KR"
	case "de":
		return "DE"
	case "fr":
		return "FR"
	case "it":
		return "IT"
	case "es":
		return "ES"
	case "pt":
		return "PT"
	case "ru":
		return "RU"
	case "tr":
		return "TR"
	case "pl":
		return "PL"
	case "nl":
		return "NL"
	case "sv":
		return "SE"
	case "no", "nb", "nn":
		return "NO"
	case "fi":
		return "FI"
	case "da":
		return "DK"
	case "cs":
		return "CZ"
	case "hu":
		return "HU"
	case "ro":
		return "RO"
	case "el":
		return "GR"
	case "th":
		return "TH"
	case "vi":
		return "VN"
	case "id":
		return "ID"
	case "ar":
		return "SA"
	case "uk":
		return "UA"
	}
	return ""
}
