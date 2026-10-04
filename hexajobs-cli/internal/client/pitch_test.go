package client

import (
	"math"
	"strings"
	"testing"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

func TestBuildPitchStructureAndEvidence(t *testing.T) {
	job := models.JobListing{Title: "Go Engineer", Company: "Acme", SkillsRequired: []string{"go", "sql"}, SkillsMatch: 0.8, CompensationUSD: 75, CompensationPeriod: "hour"}
	pitch := BuildPitch(job)
	for _, part := range []string{"Subject: Go Engineer", "Hello Acme", "go, sql", "USD 75.00/hour", "80%", "[add one project", "[Your name]", "scope"} {
		if !strings.Contains(pitch, part) {
			t.Errorf("missing %q in %s", part, pitch)
		}
	}
	if strings.Contains(pitch, "years of experience") || strings.Contains(pitch, "market average") {
		t.Fatal("invented candidate or market evidence")
	}
}

func TestBuildPitchMissingFieldsAndTerminalInjection(t *testing.T) {
	pitch := BuildPitch(models.JobListing{Title: "\x1b[31mGo\x1b[0m\nEngineer", Company: "Acme\x1b]52;c;c2VjcmV0\a", CompensationUSD: math.NaN(), SkillsMatch: math.Inf(1)})
	if strings.ContainsAny(pitch, "\x1b\a") || strings.Contains(pitch, "NaN") || strings.Contains(pitch, "Inf") {
		t.Fatal("unsafe output", pitch)
	}
	if !strings.Contains(pitch, "Hello Acme") || !strings.Contains(pitch, "budget, scope") {
		t.Fatal("missing fallbacks", pitch)
	}
	if empty := BuildPitch(models.JobListing{}); !strings.Contains(empty, "the hiring team") || !strings.Contains(empty, "this opportunity") {
		t.Fatal("empty listing not supported")
	}
	stale := BuildPitch(models.JobListing{CompensationUSD: 1000, CompensationPeriod: "task", ForexSource: "offline"})
	if !strings.Contains(stale, "USD 1000.00/task") || !strings.Contains(stale, "indicative") {
		t.Fatal("lost compensation provenance")
	}
}
