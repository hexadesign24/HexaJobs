package views

import (
	"github.com/charmbracelet/x/ansi"
)

// AppVersion is the single source of truth for the release version.
// Override at build time with:
//   -ldflags "-X hexajobs.dev/hexajobs-cli/internal/ui/views.AppVersion=v1.1"
var AppVersion = "v1.1"

// LogoBlocky is pure ASCII (32-126 only) so it survives terminal rendering
// without going through CleanText (which collapses whitespace).
// Blocky "HEXAJOBS", 5 rows x 49 cols + 1 tagline: fits 80x24 terminals.
var LogoBlocky = []string{
	`#   # ##### #   #  ###    ###  ###  ####   ####`,
	`#   # #      # #  #   #    #  #   # #   # #    `,
	`##### ####    #   #####    #  #   # ####   ### `,
	`#   # #      # #  #   # #  #  #   # #   #     #`,
	`#   # ##### #   # #   #  ##    ###  ####  #### `,
}

const LogoTagline = "HEXAJOBS.DEV - find your next move"

// ShowLogo reports whether there is enough room for the ASCII art.
// width/height are the Dashboard content box dimensions (w, h).
// Blocky art is 49 cols wide: require 52 so it never truncates.
func ShowLogo(width, height int) bool {
	if width < 52 || height < 14 {
		return false
	}
	return true
}

// Logo returns styled logo lines, each truncated to width.
// It must NOT go through Line()/Wrap()/CleanText.
func Logo(width int, s Styles) []string {
	out := make([]string, 0, len(LogoBlocky)+1)
	for _, line := range LogoBlocky {
		out = append(out, s.Title.Render(ansi.Truncate(line, max(0, width), "")))
	}
	out = append(out, s.Muted.Render(ansi.Truncate(LogoTagline, max(0, width), "")))
	return out
}
