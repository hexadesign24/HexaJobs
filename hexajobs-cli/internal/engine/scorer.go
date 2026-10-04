package engine

import (
	"math"
	"strings"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

type Profile struct {
	Skills          []string
	Text            string
	TargetHourlyUSD float64
}

// Score is an explainable ranking heuristic, not a trained hiring-probability
// model: 60% skill, 15% lexical relevance, 10% FX-normalized pay, 10% freshness,
// 5% response evidence. Risk discounts are applied before clamping to [5,98].
func Score(job models.JobListing, profile Profile, now time.Time) models.JobListing {
	job.SkillsMatch = Jaccard(profile.Skills, job.SkillsRequired)
	semantic := CosineSimilarity(profile.Text+" "+strings.Join(profile.Skills, " "), job.Title+" "+job.Description+" "+strings.Join(job.SkillsRequired, " "))
	pay := 0.5
	if hourly, ok := HourlyUSD(job); ok && finitePositive(profile.TargetHourlyUSD) {
		pay = math.Min(1, hourly/profile.TargetHourlyUSD)
	}
	freshness := 0.5
	if !job.CreatedAt.IsZero() {
		age := math.Max(0, now.Sub(job.CreatedAt).Hours()/24)
		freshness = math.Exp(-age / 30)
	}
	response := 0.5
	if job.GhostingRate >= 0 && job.GhostingRate <= 1 {
		response = 1 - job.GhostingRate
	} else {
		job.GhostingRate = -1
	}
	value := 0.60*job.SkillsMatch + 0.15*semantic + 0.10*pay + 0.10*freshness + 0.05*response
	switch job.RiskStatus {
	case models.RiskGreen:
	case models.RiskRed:
		value *= 0.5
	default:
		job.RiskStatus = models.RiskYellow
		value *= 0.85
	}
	job.WinRate = int(math.Round(5 + 93*math.Max(0, math.Min(1, value))))
	return job
}
