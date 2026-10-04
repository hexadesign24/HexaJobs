package views

import (
	"github.com/charmbracelet/x/ansi"
)

// AppVersion is the single source of truth for the release version.
// Override at build time with:
//   -ldflags "-X hexajobs.dev/hexajobs-cli/internal/ui/views.AppVersion=v1.1"
var AppVersion = "v1.0"

// LogoHexa is pure ASCII (32-126 only) so it survives terminal rendering
// without going through CleanText (which collapses whitespace).
// Keep max 5 lines + 1 tagline to fit 80x24 terminals.
var LogoHexa = []string{
	"   _____   ",
	"  /     \\  ",
	" /  HEX  \\ ",
	" \\  JOBS / ",
	"  \\_____/  ",
}

const LogoTagline = "HEXAJOBS.DEV - find your next move"

// ShowLogo reports whether there is enough room for the ASCII art.
// width/height are the Dashboard content box dimensions (w, h).
func ShowLogo(width, height int) bool {
	if width < 40 || height < 14 {
		return false
	}
	return true
}

// Logo returns styled logo lines, each truncated to width.
// It must NOT go through Line()/Wrap()/CleanText.
func Logo(width int, s Styles) []string {
	out := make([]string, 0, len(LogoHexa)+1)
	for _, line := range LogoHexa {
		out = append(out, s.Title.Render(ansi.Truncate(line, max(0, width), "")))
	}
	out = append(out, s.Muted.Render(ansi.Truncate(LogoTagline, max(0, width), "")))
	return out
}
