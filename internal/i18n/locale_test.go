package i18n

import "testing"

func TestParseLocale(t *testing.T) {
	tests := []struct {
		input string
		want  SystemLocale
		ok    bool
	}{
		{input: "zh_CN.UTF-8", want: SystemLocale{Language: "zh", Region: "CN"}, ok: true},
		{input: "en_US.UTF-8", want: SystemLocale{Language: "en", Region: "US"}, ok: true},
		{input: "en-US", want: SystemLocale{Language: "en", Region: "US"}, ok: true},
		{input: "zh-Hans-CN", want: SystemLocale{Language: "zh", Script: "Hans", Region: "CN"}, ok: true},
		{input: "zh_Hant_TW", want: SystemLocale{Language: "zh", Script: "Hant", Region: "TW"}, ok: true},
		{input: "ja_JP@japanese", want: SystemLocale{Language: "ja", Region: "JP"}, ok: true},
		{input: "pt_BR.UTF-8", want: SystemLocale{Language: "pt", Region: "BR"}, ok: true},
		{input: "ko", want: SystemLocale{Language: "ko"}, ok: true},
		{input: "C.UTF-8", ok: false},
		{input: "POSIX", ok: false},
		{input: "", ok: false},
		{input: "  ", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := parseLocale(tt.input)
			if ok != tt.ok {
				t.Fatalf("parseLocale(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if !ok {
				return
			}
			if got != tt.want {
				t.Fatalf("parseLocale(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestComputeSystemLocaleFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want SystemLocale
		ok   bool
	}{
		{
			name: "LC_ALL wins",
			env:  map[string]string{"LC_ALL": "ja_JP.UTF-8", "LC_MESSAGES": "en_US", "LANG": "en_US"},
			want: SystemLocale{Language: "ja", Region: "JP"},
			ok:   true,
		},
		{
			name: "neutral LC_ALL falls through to LANG",
			env:  map[string]string{"LC_ALL": "C.UTF-8", "LANG": "zh_CN.UTF-8"},
			want: SystemLocale{Language: "zh", Region: "CN"},
			ok:   true,
		},
		{
			name: "all neutral returns false",
			env:  map[string]string{"LC_ALL": "C", "LANG": "POSIX"},
			ok:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := computeSystemLocaleFromEnv(func(k string) string { return tt.env[k] })
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSteamLangFor(t *testing.T) {
	tests := []struct {
		loc  SystemLocale
		want string
	}{
		{loc: SystemLocale{Language: "zh", Region: "CN"}, want: "schinese"},
		{loc: SystemLocale{Language: "zh", Script: "Hans", Region: "CN"}, want: "schinese"},
		{loc: SystemLocale{Language: "zh", Region: "TW"}, want: "tchinese"},
		{loc: SystemLocale{Language: "zh", Script: "Hant", Region: "TW"}, want: "tchinese"},
		{loc: SystemLocale{Language: "zh", Region: "HK"}, want: "tchinese"},
		{loc: SystemLocale{Language: "en", Region: "US"}, want: "english"},
		{loc: SystemLocale{Language: "ja", Region: "JP"}, want: "japanese"},
		{loc: SystemLocale{Language: "ko", Region: "KR"}, want: "koreana"},
		{loc: SystemLocale{Language: "pt", Region: "BR"}, want: "brazilian"},
		{loc: SystemLocale{Language: "pt", Region: "PT"}, want: "portuguese"},
		{loc: SystemLocale{Language: "es", Region: "ES"}, want: "spanish"},
		{loc: SystemLocale{Language: "es", Region: "MX"}, want: "latam"},
		{loc: SystemLocale{Language: "es", Region: "AR"}, want: "latam"},
		{loc: SystemLocale{Language: "nb"}, want: "norwegian"},
		{loc: SystemLocale{Language: "xx"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.loc.Language+"-"+tt.loc.Region, func(t *testing.T) {
			if got := SteamLangFor(tt.loc); got != tt.want {
				t.Fatalf("SteamLangFor(%#v) = %q, want %q", tt.loc, got, tt.want)
			}
		})
	}
}

func TestSteamCCFor(t *testing.T) {
	tests := []struct {
		loc  SystemLocale
		want string
	}{
		{loc: SystemLocale{Language: "zh", Region: "CN"}, want: "CN"},
		{loc: SystemLocale{Language: "zh", Script: "Hant"}, want: "TW"},
		{loc: SystemLocale{Language: "zh"}, want: "CN"},
		{loc: SystemLocale{Language: "en"}, want: "US"},
		{loc: SystemLocale{Language: "en", Region: "GB"}, want: "GB"},
		{loc: SystemLocale{Language: "ja"}, want: "JP"},
		{loc: SystemLocale{Language: "pt", Region: "BR"}, want: "BR"},
		{loc: SystemLocale{Language: "pt"}, want: "PT"},
		{loc: SystemLocale{Language: "xx"}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.loc.Language+"-"+tt.loc.Region, func(t *testing.T) {
			if got := SteamCCFor(tt.loc); got != tt.want {
				t.Fatalf("SteamCCFor(%#v) = %q, want %q", tt.loc, got, tt.want)
			}
		})
	}
}

func TestParseLocaleConfFile(t *testing.T) {
	body := `# /etc/locale.conf
LANG="zh_CN.UTF-8"
LC_TIME="en_GB.UTF-8"
`
	loc, ok := parseLocaleConfFile(body)
	if !ok {
		t.Fatal("expected to parse LANG line")
	}
	if loc.Language != "zh" || loc.Region != "CN" {
		t.Fatalf("got %#v", loc)
	}
}

func TestDetectSystemLocaleEnvWins(t *testing.T) {
	ResetDetect()
	t.Setenv("LC_ALL", "fr_FR.UTF-8")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "")
	got := DetectSystemLocale()
	if got.Language != "fr" || got.Region != "FR" {
		t.Fatalf("got %#v", got)
	}
	ResetDetect()
}
