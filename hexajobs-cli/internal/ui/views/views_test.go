package views

import (
	"testing"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

func TestRadarUsesObservedRegionsOnly(t *testing.T) {
	jobs := []models.JobListing{{Region: "USA"}, {Region: "Australia"}, {Region: "Indonesia"}, {Region: "EU"}, {Region: "Worldwide"}, {Region: "US", IsMock: true}}
	got := RegionalCounts(jobs)
	if got["US"] != 1 || got["EU"] != 1 || got["Asia"] != 1 || got["ID"] != 1 {
		t.Fatalf("counts %v", got)
	}
}

func TestFairRateNeverMixesPeriodsOrClaimsMockAsFair(t *testing.T) {
	r := models.MarketReport{AverageRate: "USD 50.00/hour"}
	j := models.JobListing{CompensationUSD: 100000, CompensationPeriod: "year"}
	if FairRate(j, r) != "N/A" {
		t.Fatal("annual salary compared directly to hourly")
	}
	j.CompensationUSD = 60
	j.CompensationPeriod = "hour"
	if FairRate(j, r) != "at/above sample" {
		t.Fatal("same-period comparison missing")
	}
	j.ForexSource = "offline"
	if FairRate(j, r) != "indicative" {
		t.Fatal("offline rate treated as fair")
	}
}
