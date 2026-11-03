// Package banner renders the startup notice that tells the user where to open
// the installer UI and where cluster data is written.
//
// The banner is printed to stdout (not the log stream) so it stands apart from
// log noise. It is a pure string builder — no I/O — so it is trivially testable.
package banner

import (
	"os"
	"strings"
)

// ANSI escape sequences. Kept as constants so the color tests can assert on
// them without duplicating the raw codes.
const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[93m"
)

// Box-drawing characters. Rounded corners read as friendlier than square ones
// and match the ASCII style used in the project docs.
const (
	boxTL = "╭"
	boxTR = "╮"
	boxBL = "╰"
	boxBR = "╯"
	boxH  = "─"
	boxV  = "│"
)

// minWidth is the floor for the box width so short URLs still produce a
// readable banner.
const minWidth = 40

// Render returns the full banner text (box + ANSI colors) for the given URL and
// working directory. It is a pure function: no I/O, no environment reads.
func Render(url, cwd string) string {
	return render(url, cwd, true)
}

// RenderPlain returns the same banner without ANSI codes. Used when colors are
// disabled (NO_COLOR set or stdout is not a TTY).
func RenderPlain(url, cwd string) string {
	return render(url, cwd, false)
}

// SupportsColor reports whether ANSI colors should be emitted. Colors are
// suppressed when NO_COLOR is set (any value, per the no-color spec) or when
// stdout is not a TTY (CI logs, redirected output).
func SupportsColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return IsTTY()
}

// Print writes the banner to stdout with a trailing newline, choosing color vs
// plain via SupportsColor.
func Print(url, cwd string) {
	text := RenderPlain(url, cwd)
	if SupportsColor() {
		text = Render(url, cwd)
	}
	os.Stdout.WriteString(text + "\n")
}

// IsTTY reports whether stdout is a terminal. Exported as a variable so tests
// can stub it; production code never reassigns it.
var IsTTY = stdoutIsTerminal

// stdoutIsTerminal reports whether stdout is a character device (a terminal).
// Go 1.20 has no os.File.IsTerminal, so we use the ModeCharDevice heuristic:
// pipes and regular files report false, real terminals report true. The only
// false positive is /dev/null, which is harmless (codes are discarded).
func stdoutIsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func render(url, cwd string, color bool) string {
	title := "YDB Installer is ready"
	open := "Open in your browser:"
	data := "Cluster data is written to the current folder:"

	lines := []string{title, "", open, url, "", data, cwd}
	width := boxWidth(lines)

	var b strings.Builder
	b.WriteString(topLine(width))
	b.WriteString("\n")
	for _, line := range lines {
		b.WriteString(boxLine(line, width, color))
		b.WriteString("\n")
	}
	b.WriteString(bottomLine(width))
	return b.String()
}

// boxWidth returns the box inner width: the longest content line, floored at
// minWidth. ANSI codes are stripped before measuring so styled spans do not
// inflate the box.
func boxWidth(lines []string) int {
	w := minWidth
	for _, line := range lines {
		if n := displayWidth(line); n > w {
			w = n
		}
	}
	return w
}

// displayWidth returns the visible length of a line, ignoring ANSI codes.
func displayWidth(s string) int {
	return len(stripAnsi(s))
}

// stripAnsi removes \x1b[...m escape sequences from s.
func stripAnsi(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++ // skip the 'm'
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// boxLine pads a content line to the box width and wraps it in vertical
// borders. When color is on, the URL is bold green and the cwd line is dim cyan.
func boxLine(line string, width int, color bool) string {
	content := line
	if color {
		switch {
		case strings.HasPrefix(line, "http"):
			content = ansiBold + ansiGreen + line + ansiReset
		case strings.HasPrefix(line, "YDB Installer"):
			content = ansiBold + line + ansiReset
		case strings.HasPrefix(line, "Cluster data"):
			content = ansiYellow + line + ansiReset
		}
	}
	pad := width - displayWidth(content)
	return boxV + " " + content + strings.Repeat(" ", pad) + " " + boxV
}

func topLine(width int) string {
	return boxTL + strings.Repeat(boxH, width+2) + boxTR
}

func bottomLine(width int) string {
	return boxBL + strings.Repeat(boxH, width+2) + boxBR
}
