package security

import (
	"math"
	"testing"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

func TestValidateJobListing(t *testing.T) {
	for _, tc := range []struct {
		name, text, risk, want string
		pay                    float64
		period                 string
	}{
		{"ordinary", "Go engineer with paid leave", "", "GREEN", 60, "hour"},
		{"telegram", "TELEGRAM   ONLY recruitment", "", "RED", 0, ""},
		{"fee", "upfront registration fee is required", "", "RED", 0, ""},
		{"bank", "bank account rental opportunity", "", "RED", 0, ""},
		{"whatsapp", "WhatsApp interview tomorrow", "", "YELLOW", 0, ""},
		{"local scam", "sewa rekening bank", "", "RED", 0, ""},
		{"zero width", "tele\u200bgram only", "", "RED", 0, ""},
		{"preserve red", "ordinary listing", "RED", "RED", 0, ""},
		{"preserve unknown", "ordinary listing", "YELLOW", "YELLOW", 0, ""},
		{"high hourly", "Basic data entry", "", "RED", 251, "hour"},
		{"boundary hourly", "Basic data entry", "", "GREEN", 250, "hour"},
		{"annual not hourly", "Basic data entry", "", "GREEN", 60000, "year"},
		{"task not hourly", "Basic data entry", "", "GREEN", 1000, "task"},
		{"expert rate", "Security engineer", "", "GREEN", 300, "hour"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			j := models.JobListing{Title: "Position", Description: tc.text, RiskStatus: tc.risk, CompensationUSD: tc.pay, CompensationPeriod: tc.period}
			status, flags := ValidateJobListing(j)
			if status != tc.want {
				t.Fatalf("got %s want %s: %v", status, tc.want, flags)
			}
			if status != "GREEN" && len(flags) == 0 {
				t.Fatal("flagged without explanation")
			}
			if j.RiskStatus != tc.risk {
				t.Fatal("engine record mutated")
			}
		})
	}
	status, flags := ValidateJobListing(models.JobListing{IsMock: true})
	if status != "YELLOW" || len(flags) == 0 {
		t.Fatal("mock treated as verified")
	}
	status, _ = ValidateJobListing(models.JobListing{ForexSource: "offline"})
	if status != "YELLOW" {
		t.Fatal("offline FX not disclosed")
	}
}

func TestFilterGhostingStrictBoundaryAndUnknown(t *testing.T) {
	jobs := []models.JobListing{{ID: "unknown", GhostingRate: -1}, {ID: "boundary", GhostingRate: 0.75}, {ID: "high", GhostingRate: 0.75001}, {ID: "nan", GhostingRate: math.NaN()}, {ID: "certain", GhostingRate: 1}}
	got := FilterGhosting(jobs)
	if len(got) != 3 || got[0].ID != "unknown" || got[1].ID != "boundary" || got[2].ID != "nan" {
		t.Fatalf("filtered %+v", got)
	}
	got[0].ID = "changed"
	if jobs[0].ID != "unknown" {
		t.Fatal("mutated source slice")
	}
}
