package views

import "strings"

func Settings(language, region string, shield bool, cursor, width, height int, s Styles) string {
	state := Tr(language, "OFF")
	if shield {
		state = Tr(language, "ON")
	}
	rows := []string{Tr(language, "Language") + ": " + language, Tr(language, "Target region") + ": " + region, Tr(language, "ScamShield") + ": " + state}
	lines := []string{s.Title.Render(Tr(language, "Settings")), ""}
	for i, row := range rows {
		prefix := "  "
		style := s.Text
		if i == cursor {
			prefix = "> "
			style = s.Selected
		}
		lines = append(lines, style.Render(Line(prefix+row, width-2)), "")
	}
	lines = append(lines, s.Muted.Render("↑/↓ select · ←/→ or Enter change"), s.Muted.Render("Preferences save automatically to ui.json."), "", s.Muted.Render("Anti-ghosting >75% remains enabled."))
	return Box(strings.Join(lines, "\n"), width, height, s)
}
