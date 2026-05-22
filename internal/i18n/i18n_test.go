package i18n

import "testing"

func TestNormalize(t *testing.T) {
	tests := map[string]Language{
		"":         Auto,
		"auto":     Auto,
		"en":       EN,
		"english":  EN,
		"zh-CN":    ZhCN,
		"zh_Hans":  ZhCN,
		"schinese": ZhCN,
	}
	for input, want := range tests {
		if got := Normalize(input); got != want {
			t.Fatalf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSetAutoDetectsChineseEnv(t *testing.T) {
	ResetDetect()
	t.Setenv("LC_ALL", "zh_CN.UTF-8")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "")
	if got := Set("auto"); got != ZhCN {
		t.Fatalf("Set(auto) = %q, want %q", got, ZhCN)
	}
}

func TestDetectFromEnvValuesFallsBackForNeutralLocale(t *testing.T) {
	values := map[string]string{
		"LC_ALL":      "C.UTF-8",
		"LC_MESSAGES": "",
		"LANG":        "C.UTF-8",
	}
	_, ok := detectFromEnvValues(func(key string) string {
		return values[key]
	})
	if ok {
		t.Fatal("detectFromEnvValues should not decide for neutral C locale")
	}
}

func TestDetectFromTextRecognizesOSChineseLocale(t *testing.T) {
	for _, input := range []string{
		`("zh-Hans-CN")`,
		`LANG=zh_CN.UTF-8`,
		`zh-CN`,
	} {
		got, ok := detectFromText(input)
		if !ok || got != ZhCN {
			t.Fatalf("detectFromText(%q) = %q, %v; want zh-CN, true", input, got, ok)
		}
	}
}

func TestLanguageFromLocale(t *testing.T) {
	tests := []struct {
		input string
		want  Language
		ok    bool
	}{
		{input: "zh_CN.UTF-8", want: ZhCN, ok: true},
		{input: "en_US.UTF-8", want: EN, ok: true},
		{input: "C.UTF-8", ok: false},
		{input: "", ok: false},
	}
	for _, test := range tests {
		got, ok := languageFromLocale(test.input)
		if got != test.want || ok != test.ok {
			t.Fatalf("languageFromLocale(%q) = %q, %v; want %q, %v", test.input, got, ok, test.want, test.ok)
		}
	}
}

func TestTUsesFallback(t *testing.T) {
	Set("zh-CN")
	if got := T("missing.key"); got != "missing.key" {
		t.Fatalf("T(missing.key) = %q", got)
	}
	Set("en")
}

// TestKeysetParity makes sure every English key has a Chinese translation
// and vice-versa. Without this, T() silently falls back to the key string.
func TestKeysetParity(t *testing.T) {
	for key := range en {
		if _, ok := zhCN[key]; !ok {
			t.Errorf("missing zhCN translation for key %q", key)
		}
	}
	for key := range zhCN {
		if _, ok := en[key]; !ok {
			t.Errorf("zhCN has key %q not present in en (en is the canonical fallback)", key)
		}
	}
}

// TestDetectIsCached confirms repeated Set("auto") only runs the detector
// once per process. Without caching, every command would fork `defaults` or
// `powershell` on macOS/Windows.
func TestDetectIsCached(t *testing.T) {
	ResetDetect()
	t.Setenv("LC_ALL", "zh_CN.UTF-8")
	first := Set("auto")
	t.Setenv("LC_ALL", "en_US.UTF-8")
	second := Set("auto")
	if first != second {
		t.Fatalf("expected detector to cache: first=%s second=%s", first, second)
	}
	ResetDetect()
}
