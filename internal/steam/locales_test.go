package steam

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLanguageOptionsIncludeSteamLanguageCodes(t *testing.T) {
	want := map[string]bool{
		"english":   false,
		"schinese":  false,
		"koreana":   false,
		"brazilian": false,
		"latam":     false,
	}
	for _, language := range LanguageOptions() {
		if _, ok := want[language.Code]; ok {
			want[language.Code] = true
		}
	}
	for code, found := range want {
		if !found {
			t.Fatalf("LanguageOptions missing %q", code)
		}
	}
}

func TestRegionOptionsIncludeCommonPriceRegions(t *testing.T) {
	want := map[string]bool{
		"CN": false,
		"US": false,
		"JP": false,
		"GB": false,
		"DE": false,
	}
	for _, region := range RegionOptions() {
		if _, ok := want[region.Code]; ok {
			want[region.Code] = true
		}
	}
	for code, found := range want {
		if !found {
			t.Fatalf("RegionOptions missing %q", code)
		}
	}
}

func TestParseSteamStoreLanguagesHTML(t *testing.T) {
	raw := `
		<a class="popup_menu_item tight" href="?l=schinese" onclick="ChangeLanguage( 'schinese' ); return false;">简体中文 (Simplified Chinese)</a>
		<a class="popup_menu_item tight" href="?l=brazilian" onclick="ChangeLanguage( 'brazilian' ); return false;">Português-Brasil</a>
		<a class="popup_menu_item tight" href="?l=latam">Español-Latinoamérica</a>
	`
	languages := ParseSteamStoreLanguagesHTML(raw)
	want := map[string]string{
		"english":   "English",
		"schinese":  "简体中文 (Simplified Chinese)",
		"brazilian": "Português-Brasil",
		"latam":     "Español-Latinoamérica",
	}
	for _, language := range languages {
		delete(want, language.Code)
	}
	for code := range want {
		t.Fatalf("ParseSteamStoreLanguagesHTML missing %q in %#v", code, languages)
	}
}

func TestParseSteamStoreLanguagesHTMLFallsBackToBuiltIn(t *testing.T) {
	languages := ParseSteamStoreLanguagesHTML("<html></html>")
	if len(languages) != len(LanguageOptions()) {
		t.Fatalf("fallback language count = %d, want %d", len(languages), len(LanguageOptions()))
	}
}

func TestLiveLanguagesUsesClientLanguage(t *testing.T) {
	c := NewClient("US", "schinese", time.Second)
	c.HTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.Query().Get("l"); got != "schinese" {
			t.Fatalf("language query = %q, want schinese", got)
		}
		body := `<a class="popup_menu_item tight" href="?l=schinese" onclick="ChangeLanguage( 'schinese' ); return false;">简体中文</a>`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	languages, err := c.LiveLanguages()
	if err != nil {
		t.Fatalf("LiveLanguages returned error: %v", err)
	}
	if len(languages) < 2 {
		t.Fatalf("LiveLanguages returned too few options: %#v", languages)
	}
}
