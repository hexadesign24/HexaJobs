package client

import (
	"fmt"
	"math"
	"strings"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

// BuildPitch is a local draft builder: no LLM call, invented experience, or
// automatic submission. The user fills the evidence and signature placeholders.
func BuildPitch(job models.JobListing) string {
	title, company := CleanText(job.Title), CleanText(job.Company)
	if title == "" {
		title = "this opportunity"
	}
	if company == "" {
		company = "the hiring team"
	}
	skills := []string{}
	for _, s := range job.SkillsRequired {
		if s = CleanText(s); s != "" {
			skills = append(skills, s)
		}
		if len(skills) == 5 {
			break
		}
	}
	focus := "the requirements in your listing"
	if len(skills) > 0 {
		focus = strings.Join(skills, ", ")
	}
	pay := "I would welcome a discussion of the budget, scope, and delivery timeline."
	if job.CompensationUSD > 0 && !math.IsInf(job.CompensationUSD, 0) && !math.IsNaN(job.CompensationUSD) {
		period := CleanText(job.CompensationPeriod)
		if period == "" || period == "unknown" {
			period = "unspecified period"
		}
		pay = fmt.Sprintf("Your listed budget is approximately USD %.2f/%s; I would confirm the scope and target rate with you before starting.", job.CompensationUSD, period)
		if job.ForexSource == "offline" || job.ForexSource == "frankfurter-stale" {
			pay += " This conversion is indicative and needs confirmation."
		}
	}
	match := ""
	if job.SkillsMatch > 0 && job.SkillsMatch <= 1 {
		match = fmt.Sprintf("\nDraft note: engine skill alignment %.0f%%; verify the skills and evidence before sending.\n", job.SkillsMatch*100)
	}
	return fmt.Sprintf("Subject: %s — proposal\n\nHello %s,\n\nI am interested in %s. Your focus on %s caught my attention.\n\nRelevant evidence: [add one project, your contribution, and a concrete result].\n\nI propose starting with a short scope review, then agreeing on a clear first deliverable and milestone. %s\n\nWould you be open to a brief discussion about priorities and next steps?\n\nBest,\n[Your name]\n%s", title, company, title, focus, pay, match)
}
