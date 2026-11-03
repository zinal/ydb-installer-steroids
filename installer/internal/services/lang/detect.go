// Package lang provides system locale detection for the installer.
//
// The rules are intentionally simple: we look at LC_ALL, then LANG, then
// LANGUAGE (in that order). If any value contains "ru" (case-insensitive),
// the default UI language is "ru"; otherwise it falls back to "en".
package lang

import (
	"os"
	"strings"
)

// Default is the language used when no signal can be derived from the
// environment.
const Default = "en"

// EnvVarsInOrder is the precedence list used by Detect.
// Exposed for tests.
var EnvVarsInOrder = []string{"LC_ALL", "LANG", "LANGUAGE"}

// Detect returns the default UI language code based on the supplied lookup
// function. Passing os.Getenv yields the system-derived value.
func Detect(getenv func(string) string) string {
	for _, v := range EnvVarsInOrder {
		if val := getenv(v); val != "" {
			return classify(val)
		}
	}
	return Default
}

// DetectFromOS is a convenience wrapper around Detect using os.Getenv.
func DetectFromOS() string {
	return Detect(os.Getenv)
}

// classify normalises the locale string and returns "ru" if it looks Russian,
// "en" otherwise.
func classify(locale string) string {
	if locale == "" {
		return Default
	}
	low := strings.ToLower(locale)
	// Match the language tag prefix: "ru", "ru_RU.UTF-8", "ru-RU", etc.
	if strings.HasPrefix(low, "ru") || strings.Contains(low, ":ru") {
		return "ru"
	}
	return Default
}
