package steam

import (
	"html"
	"regexp"
	"strings"
)

type RegionOption struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type LanguageOption struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type RegionProbe struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	Available      bool   `json:"available"`
	Currency       string `json:"currency,omitempty"`
	FinalFormatted string `json:"final_formatted,omitempty"`
	Error          string `json:"error,omitempty"`
}

type RegionProbeResult struct {
	AppID   int           `json:"appid"`
	Source  string        `json:"source"`
	Regions []RegionProbe `json:"regions"`
}

type LocaleOptions struct {
	Regions   []RegionOption   `json:"regions"`
	Languages []LanguageOption `json:"languages"`
}

func LocaleOptionsData() LocaleOptions {
	return LocaleOptions{
		Regions:   RegionOptions(),
		Languages: LanguageOptions(),
	}
}

func RegionOptions() []RegionOption {
	return []RegionOption{
		{Code: "US", Name: "United States"},
		{Code: "CN", Name: "China"},
		{Code: "HK", Name: "Hong Kong"},
		{Code: "TW", Name: "Taiwan"},
		{Code: "JP", Name: "Japan"},
		{Code: "KR", Name: "South Korea"},
		{Code: "SG", Name: "Singapore"},
		{Code: "TH", Name: "Thailand"},
		{Code: "VN", Name: "Vietnam"},
		{Code: "ID", Name: "Indonesia"},
		{Code: "MY", Name: "Malaysia"},
		{Code: "PH", Name: "Philippines"},
		{Code: "IN", Name: "India"},
		{Code: "AU", Name: "Australia"},
		{Code: "NZ", Name: "New Zealand"},
		{Code: "GB", Name: "United Kingdom"},
		{Code: "DE", Name: "Germany"},
		{Code: "FR", Name: "France"},
		{Code: "IT", Name: "Italy"},
		{Code: "ES", Name: "Spain"},
		{Code: "NL", Name: "Netherlands"},
		{Code: "BE", Name: "Belgium"},
		{Code: "AT", Name: "Austria"},
		{Code: "CH", Name: "Switzerland"},
		{Code: "SE", Name: "Sweden"},
		{Code: "NO", Name: "Norway"},
		{Code: "DK", Name: "Denmark"},
		{Code: "FI", Name: "Finland"},
		{Code: "PL", Name: "Poland"},
		{Code: "CZ", Name: "Czechia"},
		{Code: "HU", Name: "Hungary"},
		{Code: "RO", Name: "Romania"},
		{Code: "TR", Name: "Turkey"},
		{Code: "UA", Name: "Ukraine"},
		{Code: "BR", Name: "Brazil"},
		{Code: "MX", Name: "Mexico"},
		{Code: "AR", Name: "Argentina"},
		{Code: "CL", Name: "Chile"},
		{Code: "CO", Name: "Colombia"},
		{Code: "PE", Name: "Peru"},
		{Code: "CA", Name: "Canada"},
		{Code: "ZA", Name: "South Africa"},
		{Code: "SA", Name: "Saudi Arabia"},
		{Code: "AE", Name: "United Arab Emirates"},
	}
}

func LanguageOptions() []LanguageOption {
	return []LanguageOption{
		{Code: "english", Name: "English"},
		{Code: "schinese", Name: "Simplified Chinese"},
		{Code: "tchinese", Name: "Traditional Chinese"},
		{Code: "japanese", Name: "Japanese"},
		{Code: "koreana", Name: "Korean"},
		{Code: "thai", Name: "Thai"},
		{Code: "bulgarian", Name: "Bulgarian"},
		{Code: "czech", Name: "Czech"},
		{Code: "danish", Name: "Danish"},
		{Code: "german", Name: "German"},
		{Code: "spanish", Name: "Spanish - Spain"},
		{Code: "latam", Name: "Spanish - Latin America"},
		{Code: "greek", Name: "Greek"},
		{Code: "french", Name: "French"},
		{Code: "italian", Name: "Italian"},
		{Code: "indonesian", Name: "Indonesian"},
		{Code: "hungarian", Name: "Hungarian"},
		{Code: "dutch", Name: "Dutch"},
		{Code: "norwegian", Name: "Norwegian"},
		{Code: "polish", Name: "Polish"},
		{Code: "portuguese", Name: "Portuguese - Portugal"},
		{Code: "brazilian", Name: "Portuguese - Brazil"},
		{Code: "romanian", Name: "Romanian"},
		{Code: "russian", Name: "Russian"},
		{Code: "finnish", Name: "Finnish"},
		{Code: "swedish", Name: "Swedish"},
		{Code: "turkish", Name: "Turkish"},
		{Code: "vietnamese", Name: "Vietnamese"},
		{Code: "ukrainian", Name: "Ukrainian"},
		{Code: "arabic", Name: "Arabic"},
	}
}

func ParseSteamStoreLanguagesHTML(raw string) []LanguageOption {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?is)<a[^>]+onclick="[^"]*ChangeLanguage\(\s*'([^']+)'\s*\)[^"]*"[^>]*>(.*?)</a>`),
		regexp.MustCompile(`(?is)<a[^>]+href="\?l=([^"&]+)"[^>]*>(.*?)</a>`),
	}
	seen := map[string]bool{}
	out := []LanguageOption{}
	add := func(code, name string) {
		code = strings.TrimSpace(code)
		name = cleanHTMLText(name)
		if code == "" || name == "" || seen[code] {
			return
		}
		seen[code] = true
		out = append(out, LanguageOption{Code: code, Name: name})
	}
	for _, pattern := range patterns {
		for _, match := range pattern.FindAllStringSubmatch(raw, -1) {
			add(match[1], match[2])
		}
	}
	if !seen["english"] {
		out = append([]LanguageOption{{Code: "english", Name: "English"}}, out...)
	}
	if len(out) == 1 && out[0].Code == "english" {
		return LanguageOptions()
	}
	return out
}

func cleanHTMLText(value string) string {
	value = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(value, "")
	value = html.UnescapeString(value)
	return strings.Join(strings.Fields(value), " ")
}
