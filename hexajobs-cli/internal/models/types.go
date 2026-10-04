// Package models defines the backend/UI boundary. It has no presentation dependencies.
package models

import (
	"context"
	"time"
)

const (
	CategoryIT     = "IT"
	CategoryNonIT  = "Non-IT"
	CategoryWeb3   = "Web3"
	CategoryBounty = "Bounty"
	RiskGreen      = "GREEN"
	RiskYellow     = "YELLOW"
	RiskRed        = "RED"
)

type JobListing struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Company         string    `json:"company"`
	Region          string    `json:"region"`
	Category        string    `json:"category"`
	CompensationRaw string    `json:"compensation_raw"`
	CompensationUSD float64   `json:"compensation_usd"`
	ApplyURL        string    `json:"apply_url"`
	SkillsRequired  []string  `json:"skills_required"`
	SkillsMatch     float64   `json:"skills_match"` // [0,1]; unknown requirements yield zero.
	WinRate         int       `json:"win_rate"`     // [5,98]; heuristic, not a calibrated probability.
	RiskStatus      string    `json:"risk_status"`
	GhostingRate    float64   `json:"ghosting_rate"` // [0,1], or -1 when no evidence exists.
	Platform        string    `json:"platform"`
	CreatedAt       time.Time `json:"created_at"`

	// Additive provenance fields prevent mocks, unknown pay, and offline FX being
	// mistaken for verified data. USD retains the original compensation period.
	Description        string  `json:"description,omitempty"`
	CompensationAmount float64 `json:"compensation_amount,omitempty"`
	Currency           string  `json:"currency,omitempty"`
	CompensationPeriod string  `json:"compensation_period,omitempty"` // hour, day, week, month, year, task, unknown
	ForexSource        string  `json:"forex_source,omitempty"`        // native, frankfurter, frankfurter-stale, offline, unsupported
	IsMock             bool    `json:"is_mock,omitempty"`
	AuditStatus        string  `json:"audit_status,omitempty"` // passed, failed, unknown; feed assertion only
}

type MarketReport struct {
	Sector       string   `json:"sector"`
	DemandGrowth float64  `json:"demand_growth"` // percentage points of change vs previous 7-day window
	AverageRate  string   `json:"average_rate"`
	Sentiment    string   `json:"sentiment"`
	TopSkills    []string `json:"top_skills"`
}

type EngineContract interface {
	FetchJobs(ctx context.Context, keyword string, region string, category string) ([]JobListing, error)
	GetMarketPulse(sector string) MarketReport
	PurgeExpiredCache() int
}
