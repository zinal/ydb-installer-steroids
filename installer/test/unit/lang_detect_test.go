package unit_test

import (
	"testing"

	langdetect "ydb-installer/internal/services/lang"
)

func TestDetect_DefaultsToEnglish(t *testing.T) {
	getenv := func(string) string { return "" }
	if got := langdetect.Detect(getenv); got != "en" {
		t.Fatalf("expected en for empty env, got %q", got)
	}
}

func TestDetect_RussianFromLang(t *testing.T) {
	cases := []string{"ru", "ru_RU", "ru_RU.UTF-8", "RU_ru.utf8"}
	for _, c := range cases {
		c := c
		t.Run(c, func(t *testing.T) {
			getenv := mockEnv(map[string]string{"LANG": c})
			if got := langdetect.Detect(getenv); got != "ru" {
				t.Fatalf("LANG=%q → expected ru, got %q", c, got)
			}
		})
	}
}

func TestDetect_NonRussianFromLang(t *testing.T) {
	for _, c := range []string{"en_US.UTF-8", "de_DE", "fr_FR"} {
		c := c
		t.Run(c, func(t *testing.T) {
			getenv := mockEnv(map[string]string{"LANG": c})
			if got := langdetect.Detect(getenv); got != "en" {
				t.Fatalf("LANG=%q → expected en, got %q", c, got)
			}
		})
	}
}

func TestDetect_LCAllOverridesLang(t *testing.T) {
	// LC_ALL is checked before LANG; setting LC_ALL=en should win even if LANG=ru.
	getenv := mockEnv(map[string]string{
		"LC_ALL": "en_US.UTF-8",
		"LANG":   "ru_RU.UTF-8",
	})
	if got := langdetect.Detect(getenv); got != "en" {
		t.Fatalf("expected en (LC_ALL precedence), got %q", got)
	}
}

func TestDetect_LanguageFallback(t *testing.T) {
	getenv := mockEnv(map[string]string{"LANGUAGE": "ru:en"})
	if got := langdetect.Detect(getenv); got != "ru" {
		t.Fatalf("expected ru via LANGUAGE, got %q", got)
	}
}

func mockEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}
