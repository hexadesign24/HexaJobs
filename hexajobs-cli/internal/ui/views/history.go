package views

import (
	"fmt"
	"strings"

	"hexajobs.dev/hexajobs-cli/internal/client"
)

func History(entries []client.HistoryEntry, cursor, width, height int, language string, s Styles) string {
	lines := []string{s.Title.Render(Tr(language, "Local history")), ""}
	if len(entries) == 0 {
		lines = append(lines, s.Muted.Render(Tr(language, "No history yet")))
	} else {
		capacity := max(1, (height-5)/3)
		start := cursor / capacity * capacity
		for i := start; i < min(len(entries), start+capacity); i++ {
			entry := entries[i]
			prefix := "  "
			style := s.Text
			if i == cursor {
				prefix = "> "
				style = s.Title
			}
			lines = append(lines, style.Render(Line(prefix+entry.Job.Title, width-2)), s.Muted.Render(Line(fmt.Sprintf("  %s · %s · %s", entry.At.Local().Format("2006-01-02 15:04"), entry.Action, entry.Job.Platform), width-2)), "")
		}
	}
	return Box(strings.Join(lines, "\n"), width, height, s)
}
