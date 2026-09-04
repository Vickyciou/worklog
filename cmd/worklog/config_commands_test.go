package main

import "testing"

func TestNormalizeLanguage(t *testing.T) {
	tests := map[string]string{
		"zh_TW.UTF-8": "zh-TW",
		"en-US":       "en-US",
		"ja_JP@utf8":  "ja-JP",
		"C":           "",
		"POSIX":       "",
		"":            "",
	}
	for input, expected := range tests {
		if actual := normalizeLanguage(input); actual != expected {
			t.Errorf("normalizeLanguage(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestValidTimezone(t *testing.T) {
	for _, timezone := range []string{"UTC", "Asia/Taipei"} {
		if actual, valid := validTimezone(timezone); !valid || actual != timezone {
			t.Errorf("validTimezone(%q) = %q, %t", timezone, actual, valid)
		}
	}
	for _, timezone := range []string{"", "Local", "Not/A-Timezone"} {
		if actual, valid := validTimezone(timezone); valid || actual != "" {
			t.Errorf("validTimezone(%q) = %q, %t; want empty, false", timezone, actual, valid)
		}
	}
}

func TestDetectLanguageUsesLocalePrecedence(t *testing.T) {
	t.Setenv("LC_ALL", "zh_TW.UTF-8")
	t.Setenv("LC_MESSAGES", "ja_JP.UTF-8")
	t.Setenv("LANG", "en_US.UTF-8")

	if actual, detected := detectLanguage(); !detected || actual != "zh-TW" {
		t.Fatalf("detectLanguage() = %q, %t; want zh-TW, true", actual, detected)
	}
}

func TestDetectLanguageRejectsNeutralLocale(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	t.Setenv("LC_MESSAGES", "ja_JP.UTF-8")
	t.Setenv("LANG", "en_US.UTF-8")

	if actual, detected := detectLanguage(); detected || actual != "" {
		t.Fatalf("detectLanguage() = %q, %t; want empty, false", actual, detected)
	}
}
