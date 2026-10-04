package client

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// CleanText strips terminal escape sequences and directional/control characters
// from externally supplied fields. Styles must only be added after this step.
func CleanText(text string) string {
	text = ansi.Strip(text)
	text = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) {
			return -1
		}
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
	return strings.Join(strings.Fields(text), " ")
}
