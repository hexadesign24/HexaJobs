package views

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"hexajobs.dev/hexajobs-cli/internal/client"
	"hexajobs.dev/hexajobs-cli/internal/models"
	"hexajobs.dev/hexajobs-cli/internal/security"
)

func Compensation(job models.JobListing) string {
	if job.CompensationUSD > 0 && !math.IsNaN(job.CompensationUSD) && !math.IsInf(job.CompensationUSD, 0) {
		period := job.CompensationPeriod
		if period == "" {
			period = "unknown"
		}
		return fmt.Sprintf("USD %.2f/%s", job.CompensationUSD, client.CleanText(period))
	}
	if job.CompensationRaw != "" {
		return client.CleanText(job.CompensationRaw)
	}
	return "Pay not disclosed"
}

// FairRate compares amounts only when the engine's report uses the same period.
// No FX, annualization, salary prediction or scoring formula lives in the UI.
func FairRate(job models.JobListing, report models.MarketReport) string {
	if job.IsMock || job.ForexSource == "offline" || job.ForexSource == "frankfurter-stale" {
		return "indicative"
	}
	parts := strings.Split(strings.TrimPrefix(report.AverageRate, "USD "), "/")
	if len(parts) != 2 || parts[1] != job.CompensationPeriod || job.CompensationUSD <= 0 {
		return "N/A"
	}
	rate, err := strconv.ParseFloat(parts[0], 64)
	if err != nil || rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return "N/A"
	}
	if job.CompensationUSD >= rate {
		return "at/above sample"
	}
	return "below sample"
}

func ShieldBadge(job models.JobListing, enabled bool, s Styles) string {
	if !enabled {
		return s.Muted.Render("Shield OFF")
	}
	status, _ := security.ValidateJobListing(job)
	switch status {
	case models.RiskRed:
		return s.Red.Render("🔴 Risky")
	case models.RiskYellow:
		return s.Yellow.Render("🟡 Caution")
	default:
		return s.Green.Render("🟢 Verified*")
	}
}

func Results(jobs []models.JobListing, cursor int, report models.MarketReport, shield bool, width, height int, language string, s Styles) string {
	if len(jobs) == 0 {
		return Box(s.Title.Render(Tr(language, "Results"))+"\n\n"+Tr(language, "No listings to show")+"\n\n"+s.Muted.Render("Esc: "+Tr(language, "Back")), width, height, s)
	}
	capacity := max(1, (height-5)/5)
	start := (cursor / capacity) * capacity
	end := min(len(jobs), start+capacity)
	lines := []string{s.Title.Render(fmt.Sprintf("%s %d/%d", Tr(language, "Results"), cursor+1, len(jobs))), ""}
	for i := start; i < end; i++ {
		job := jobs[i]
		prefix := "  "
		style := s.Text
		if i == cursor {
			prefix = "> "
			style = s.Title
		}
		mock := ""
		if job.IsMock {
			mock = " [DEMO]"
		}
		lines = append(lines, style.Render(Line(prefix+job.Title+mock, width-2)), s.Muted.Render(Line("  "+job.Company+" · "+job.Region+" · "+job.Platform, width-2)), Line("  "+Compensation(job)+" | Fair: "+FairRate(job, report), width-2), fmt.Sprintf("  Win %d%%  %s", job.WinRate, ShieldBadge(job, shield, s)), "")
	}
	return Box(strings.Join(lines, "\n"), width, height, s)
}

func Inspect(job models.JobListing, report models.MarketReport, shield bool) string {
	status, flags := security.ValidateJobListing(job)
	if !shield {
		status += " (protection disabled)"
	}
	lines := []string{job.Title, job.Company + " · " + job.Region, "Source: " + job.Platform, "", Compensation(job) + " | Fair rate: " + FairRate(job, report), fmt.Sprintf("Win-rate: %d%% (heuristic) | Skill match: %.0f%%", job.WinRate, job.SkillsMatch*100), "ScamShield: " + status}
	for _, flag := range flags {
		lines = append(lines, "- "+flag)
	}
	ghost := "Unknown"
	if job.GhostingRate >= 0 && job.GhostingRate <= 1 {
		ghost = fmt.Sprintf("%.0f%%", job.GhostingRate*100)
	}
	lines = append(lines, "Ghosting: "+ghost, "", "Skills: "+strings.Join(job.SkillsRequired, ", "), "", job.Description, "", "Link: "+job.ApplyURL, "", "Verified = no heuristic red flag; not an identity or contract audit.")
	for i, line := range lines {
		lines[i] = client.CleanText(line)
	}
	return strings.Join(lines, "\n")
}
