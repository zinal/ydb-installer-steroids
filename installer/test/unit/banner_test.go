package unit_test

import (
	"strings"
	"testing"

	"ydb-installer/internal/banner"
)

const (
	testURL = "http://127.0.0.1:8080"
	testCwd = "/srv/cluster-a"
)

// TestBanner_ContainsURL verifies the rendered banner includes the URL.
func TestBanner_ContainsURL(t *testing.T) {
	got := banner.RenderPlain(testURL, testCwd)
	if !strings.Contains(got, testURL) {
		t.Errorf("banner missing URL %q:\n%s", testURL, got)
	}
}

// TestBanner_ContainsCwd verifies the banner tells the user where cluster data
// is written (the working directory, not the binary's folder).
func TestBanner_ContainsCwd(t *testing.T) {
	got := banner.RenderPlain(testURL, testCwd)
	if !strings.Contains(got, testCwd) {
		t.Errorf("banner missing working directory %q:\n%s", testCwd, got)
	}
	if !strings.Contains(got, "Cluster data is written to the current folder") {
		t.Error("banner missing the working-directory notice")
	}
}

// TestBanner_HasBoxBorders verifies the banner is wrapped in a box.
func TestBanner_HasBoxBorders(t *testing.T) {
	got := banner.RenderPlain(testURL, testCwd)
	if !strings.HasPrefix(got, "╭") {
		t.Errorf("banner must start with a top-left corner:\n%s", got)
	}
	if !strings.HasSuffix(got, "╯") {
		t.Errorf("banner must end with a bottom-right corner:\n%s", got)
	}
	if !strings.Contains(got, "│") {
		t.Error("banner must contain vertical borders")
	}
}

// TestBanner_PlainHasNoAnsi verifies the plain renderer emits no escape codes.
func TestBanner_PlainHasNoAnsi(t *testing.T) {
	got := banner.RenderPlain(testURL, testCwd)
	if strings.Contains(got, "\x1b[") {
		t.Errorf("plain banner must not contain ANSI codes:\n%q", got)
	}
}

// TestBanner_ColorHasAnsi verifies the color renderer emits bold-green for the
// URL and bold for the title.
func TestBanner_ColorHasAnsi(t *testing.T) {
	got := banner.Render(testURL, testCwd)
	if !strings.Contains(got, "\x1b[") {
		t.Error("color banner must contain ANSI codes")
	}
	if !strings.Contains(got, "\x1b[1m\x1b[32m"+testURL+"\x1b[0m") {
		t.Errorf("URL must be bold green:\n%q", got)
	}
	if !strings.Contains(got, "\x1b[1mYDB Installer is ready\x1b[0m") {
		t.Error("title must be bold")
	}
}

// TestBanner_NoColorEnvDisablesColor verifies NO_COLOR suppresses color even
// when stdout is a terminal.
func TestBanner_NoColorEnvDisablesColor(t *testing.T) {
	old := banner.IsTTY
	banner.IsTTY = func() bool { return true }
	defer func() { banner.IsTTY = old }()

	t.Setenv("NO_COLOR", "1")
	if banner.SupportsColor() {
		t.Error("SupportsColor must be false when NO_COLOR is set")
	}
}

// TestBanner_NonTTYDisablesColor verifies color is suppressed when stdout is
// not a terminal.
func TestBanner_NonTTYDisablesColor(t *testing.T) {
	old := banner.IsTTY
	banner.IsTTY = func() bool { return false }
	defer func() { banner.IsTTY = old }()

	t.Setenv("NO_COLOR", "")
	if banner.SupportsColor() {
		t.Error("SupportsColor must be false when stdout is not a TTY")
	}
}

// TestBanner_BoxWidthScales verifies a URL longer than the minWidth floor
// produces a wider box.
func TestBanner_BoxWidthScales(t *testing.T) {
	short := banner.RenderPlain("http://a:1", testCwd)
	long := banner.RenderPlain("http://very-long-hostname-that-exceeds-the-minimum-width.example.com:8080", testCwd)
	if len(long) <= len(short) {
		t.Error("longer URL must produce a wider banner")
	}
}
