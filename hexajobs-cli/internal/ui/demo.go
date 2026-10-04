package ui

import (
	"context"
	"strings"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

// DemoEngine supplies clearly marked fixtures for trying the TUI without network
// access. It does not implement ingestion, scoring, or market estimation.
type DemoEngine struct{}

func (DemoEngine) PurgeExpiredCache() int { return 0 }
func (DemoEngine) GetMarketPulse(sector string) models.MarketReport {
	return models.MarketReport{Sector: sector, AverageRate: "USD 65.00/hour", Sentiment: "STABLE", TopSkills: []string{"go", "sql", "solidity"}}
}
func (DemoEngine) FetchJobs(ctx context.Context, keyword, region, category string) ([]models.JobListing, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	jobs := []models.JobListing{
		{ID: "demo-go", Title: "Go Backend Engineer", Company: "Example Labs", Region: "Worldwide", Category: "IT", CompensationRaw: "USD 65/hour", CompensationUSD: 65, CompensationPeriod: "hour", ApplyURL: "https://example.com/go", SkillsRequired: []string{"go", "sql", "docker"}, SkillsMatch: 0.8, WinRate: 82, RiskStatus: "YELLOW", GhostingRate: -1, Description: "Build reliable Go services. Demonstration listing only."},
		{ID: "demo-design", Title: "Product Designer", Company: "Example Studio", Region: "EU", Category: "Non-IT", CompensationUSD: 55, CompensationPeriod: "hour", ApplyURL: "https://example.com/design", SkillsRequired: []string{"design"}, SkillsMatch: 0.4, WinRate: 61, RiskStatus: "YELLOW", GhostingRate: 0.2},
		{ID: "demo-web3", Title: "Solidity Research Engineer", Company: "Example Protocol", Region: "Asia", Category: "Web3", CompensationUSD: 80, CompensationPeriod: "hour", ApplyURL: "https://example.com/web3", SkillsRequired: []string{"solidity"}, SkillsMatch: 0.7, WinRate: 74, RiskStatus: "YELLOW", GhostingRate: -1},
		{ID: "demo-bounty", Title: "Go test coverage bounty", Company: "Example OSS", Region: "Worldwide", Category: "Bounty", CompensationUSD: 350, CompensationPeriod: "task", ApplyURL: "https://example.com/bounty", SkillsRequired: []string{"go"}, SkillsMatch: 0.9, WinRate: 86, RiskStatus: "YELLOW", GhostingRate: -1},
	}
	out := []models.JobListing{}
	for _, job := range jobs {
		if category != "" && job.Category != category {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(job.Title+" "+strings.Join(job.SkillsRequired, " ")), strings.ToLower(keyword)) {
			continue
		}
		job.IsMock = true
		job.Platform = "demo"
		job.ForexSource = "native"
		job.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		out = append(out, job)
	}
	return out, nil
}
