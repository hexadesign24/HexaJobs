package views

import (
	"fmt"
	"strings"
	"unicode"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

// RegionalCounts is presentation grouping of observed listings, not a market
// forecast. ID is also part of Asia; worldwide records are not assigned to all.
func RegionalCounts(jobs []models.JobListing) map[string]int {
	counts := map[string]int{"US": 0, "EU": 0, "Asia": 0, "ID": 0}
	for _, job := range jobs {
		if job.IsMock {
			continue
		}
		r := " " + strings.Join(strings.FieldsFunc(strings.ToLower(job.Region), func(r rune) bool { return !unicode.IsLetter(r) }), " ") + " "
		match := func(terms ...string) bool {
			for _, term := range terms {
				if strings.Contains(r, " "+term+" ") {
					return true
				}
			}
			return false
		}
		if match("us", "usa", "united states") {
			counts["US"]++
		}
		if match("eu", "europe", "european union", "germany", "france", "netherlands", "spain", "italy", "poland", "portugal", "sweden", "ireland", "austria", "belgium", "finland", "denmark", "czechia", "romania", "greece") {
			counts["EU"]++
		}
		id := match("id", "indonesia")
		if id {
			counts["ID"]++
		}
		if id || match("asia", "apac", "japan", "singapore", "india", "china", "thailand", "vietnam", "philippines", "malaysia", "korea", "taiwan") {
			counts["Asia"]++
		}
	}
	return counts
}

func Radar(jobs []models.JobListing, report models.MarketReport, width, height int, language string, s Styles) string {
	counts := RegionalCounts(jobs)
	peak := 1
	for _, n := range counts {
		peak = max(peak, n)
	}
	barWidth := min(24, max(4, width-18))
	lines := []string{s.Title.Render(Tr(language, "Regional radar")), s.Muted.Render(Tr(language, "Observed sample")), ""}
	for _, region := range []string{"US", "EU", "Asia", "ID"} {
		n := counts[region]
		filled := barWidth * n / peak
		bar := s.Title.Render(strings.Repeat("█", filled)) + s.Muted.Render(strings.Repeat("░", barWidth-filled))
		lines = append(lines, fmt.Sprintf("%-4s %s %4d", region, bar, n), "")
	}
	rate := report.AverageRate
	if rate == "" {
		rate = "N/A"
	}
	lines = append(lines, Line("Market sample: "+rate, width-2), s.Muted.Render("ID overlaps Asia; global/unknown excluded."), s.Muted.Render("R: refresh · no regional-rate API available"))
	return Box(strings.Join(lines, "\n"), width, height, s)
}
