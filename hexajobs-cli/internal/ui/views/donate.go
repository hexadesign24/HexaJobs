package views

import "strings"

func Donate(sponsorURL string, width, height int, language string, s Styles) string {
	lines := []string{s.Title.Render(Tr(language, "Support open source")), "", Wrap("hexajobs.dev is built for people finding their next opportunity. Thank you for helping improve it.", width-2), "", Wrap("Useful contributions: reproducible bug reports, translations, accessibility checks, and pull requests.", width-2), ""}
	if sponsorURL == "" {
		lines = append(lines, s.Muted.Render("Sponsor link is not configured."))
	} else {
		lines = append(lines, Wrap(sponsorURL, width-2), s.Green.Render("A: open sponsor page"))
	}
	return Box(strings.Join(lines, "\n"), width, height, s)
}
