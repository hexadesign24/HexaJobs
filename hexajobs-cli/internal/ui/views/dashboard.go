package views

import (
	"fmt"
	"strings"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

func Toolbar(width int, engineStatus, region, language string, s Styles) string {
	return s.Title.Render(Line("HEXAJOBS.DEV [v1.0]", width)) + "\n" + s.Muted.Render(Line(fmt.Sprintf("ENGINE %s  /  REGION %s  /  %s", engineStatus, strings.ToUpper(region), strings.ToUpper(language)), width))
}

func Dashboard(input string, pulse models.MarketReport, width, height int, language string, s Styles) string {
	lines := []string{s.Title.Render(Tr(language, "FIND YOUR NEXT MOVE")), "", Tr(language, "Search jobs"), input, s.Muted.Render(Tr(language, "Press Enter to search")), "", s.Title.Render(Tr(language, "Live Market Pulse"))}
	if pulse.AverageRate == "" || pulse.AverageRate == "N/A" {
		lines = append(lines, s.Muted.Render(Tr(language, "No market data yet")))
	} else {
		lines = append(lines, Line(pulse.Sentiment+"  /  "+pulse.AverageRate, width-2), fmt.Sprintf("7d demand change: %+.1f%%", pulse.DemandGrowth), Line("Skills: "+strings.Join(pulse.TopSkills, ", "), width-2))
	}
	lines = append(lines, "", s.Muted.Render(Tr(language, "Observed sample")+" · win-rate is a heuristic"))
	return Box(strings.Join(lines, "\n"), width, height, s)
}
