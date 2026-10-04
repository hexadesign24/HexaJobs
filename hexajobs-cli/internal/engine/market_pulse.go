package engine

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

// MarketPulse retains at most 20,000 distinct, non-mock listings over 14 days.
// Reports describe the observed sample, not the entire labor market.
type MarketPulse struct {
	mu   sync.Mutex
	jobs map[string]models.JobListing
	now  func() time.Time
}

func NewMarketPulse(now func() time.Time) *MarketPulse {
	if now == nil {
		now = time.Now
	}
	return &MarketPulse{jobs: make(map[string]models.JobListing), now: now}
}

// Snapshot returns independent records for persistence by the core cache.
func (p *MarketPulse) Snapshot() []models.JobListing {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.prune(p.now())
	jobs := make([]models.JobListing, 0, len(p.jobs))
	for _, job := range p.jobs {
		job.SkillsRequired = append([]string(nil), job.SkillsRequired...)
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].Platform+jobs[i].ID < jobs[j].Platform+jobs[j].ID })
	return jobs
}

func (p *MarketPulse) Observe(jobs []models.JobListing) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	for _, job := range jobs {
		if job.ID == "" || job.IsMock || job.CreatedAt.IsZero() || job.CreatedAt.After(now) || !job.CreatedAt.After(now.Add(-14*24*time.Hour)) {
			continue
		}
		job.SkillsRequired = append([]string(nil), job.SkillsRequired...)
		// Reports need numerical fields and skills, not full source descriptions.
		job.Description = ""
		job.CompensationRaw = ""
		p.jobs[job.Platform+":"+job.ID] = job
	}
	p.prune(now)
}

func (p *MarketPulse) prune(now time.Time) {
	for key, job := range p.jobs {
		if !job.CreatedAt.After(now.Add(-14 * 24 * time.Hour)) {
			delete(p.jobs, key)
		}
	}
	if len(p.jobs) <= 20000 {
		return
	}
	keys := make([]string, 0, len(p.jobs))
	for key := range p.jobs {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := p.jobs[keys[i]], p.jobs[keys[j]]
		if a.CreatedAt.Equal(b.CreatedAt) {
			return keys[i] < keys[j]
		}
		return a.CreatedAt.Before(b.CreatedAt)
	})
	for _, key := range keys[:len(keys)-20000] {
		delete(p.jobs, key)
	}
}

func (p *MarketPulse) Report(sector string) models.MarketReport {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	p.prune(now)
	report := models.MarketReport{Sector: sector, AverageRate: "N/A", Sentiment: "STABLE", TopSkills: []string{}}
	var previous, current, hourlyCount, taskCount int
	var hourlyTotal, taskTotal float64
	skills := make(map[string]int)
	for _, job := range p.jobs {
		if sector != "" && !strings.EqualFold(sector, "all") && !strings.EqualFold(job.Category, sector) {
			continue
		}
		if job.CreatedAt.After(now.Add(-7 * 24 * time.Hour)) {
			current++
			if rate, ok := HourlyUSD(job); ok {
				hourlyTotal += rate
				hourlyCount++
			} else if job.CompensationPeriod == "task" && finitePositive(job.CompensationUSD) {
				taskTotal += job.CompensationUSD
				taskCount++
			}
			for skill := range skillSet(job.SkillsRequired) {
				skills[skill]++
			}
		} else {
			previous++
		}
	}
	// A zero baseline is insufficient evidence for a percentage growth estimate.
	if previous > 0 {
		report.DemandGrowth = 100 * float64(current-previous) / float64(previous)
		if report.DemandGrowth > 5 {
			report.Sentiment = "BULLISH"
		} else if report.DemandGrowth < -5 {
			report.Sentiment = "BEARISH"
		}
	}
	if hourlyCount > 0 {
		report.AverageRate = fmt.Sprintf("USD %.2f/hour", hourlyTotal/float64(hourlyCount))
	} else if taskCount > 0 {
		report.AverageRate = fmt.Sprintf("USD %.2f/task", taskTotal/float64(taskCount))
	}
	for skill := range skills {
		report.TopSkills = append(report.TopSkills, skill)
	}
	sort.Slice(report.TopSkills, func(i, j int) bool {
		a, b := report.TopSkills[i], report.TopSkills[j]
		if skills[a] == skills[b] {
			return a < b
		}
		return skills[a] > skills[b]
	})
	if len(report.TopSkills) > 5 {
		report.TopSkills = report.TopSkills[:5]
	}
	return report
}
