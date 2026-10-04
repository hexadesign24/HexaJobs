package engine

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

func TestSimilarity(t *testing.T) {
	for _, tc := range []struct {
		name string
		a, b []string
		want float64
	}{
		{"aliases and duplicates", []string{" Go ", "golang", "SQL"}, []string{"go", "docker"}, 1.0 / 3},
		{"empty", nil, nil, 0}, {"disjoint", []string{"c++"}, []string{"c#"}, 0}, {"exact", []string{"K8s"}, []string{"kubernetes"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Jaccard(tc.a, tc.b); math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
	if CosineSimilarity("", "") != 0 || CosineSimilarity("go sql", "python") != 0 {
		t.Fatal("empty or disjoint cosine must be zero")
	}
	if got := CosineSimilarity("Go SQL SQL", "golang sql sql"); math.Abs(got-1) > 1e-9 {
		t.Fatalf("alias cosine=%v", got)
	}
	if got := CosineSimilarity("go go sql", "go sql"); got <= 0 || got >= 1 {
		t.Fatalf("frequency cosine=%v", got)
	}
}

type getterFunc func(context.Context, string, map[string]string, any) error

func (f getterFunc) GetJSON(c context.Context, u string, h map[string]string, v any) error {
	return f(c, u, h, v)
}

func TestCurrencyRefreshConcurrencyAndFallback(t *testing.T) {
	now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	var calls atomic.Int32
	getter := getterFunc(func(_ context.Context, _ string, _ map[string]string, v any) error {
		calls.Add(1)
		return json.Unmarshal([]byte(`{"base":"USD","date":"2026-10-02","rates":{"EUR":0.8,"JPY":150,"IDR":16000}}`), v)
	})
	fx := NewCurrencyConverter(getter, "test", func() time.Time { return now })
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fx.Refresh(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("expected one refresh, got %d", calls.Load())
	}
	for _, tc := range []struct {
		amount   float64
		currency string
		want     float64
	}{{80, "EUR", 100}, {15000, "JPY", 100}, {1600000, "IDR", 100}, {100, "USD", 100}} {
		got, err := fx.ToUSD(tc.amount, tc.currency)
		if err != nil || math.Abs(got-tc.want) > 1e-9 {
			t.Fatalf("%s: %v %v", tc.currency, got, err)
		}
	}
	for _, amount := range []float64{-1, math.NaN(), math.Inf(1)} {
		if _, err := fx.ToUSD(amount, "USD"); err == nil {
			t.Fatal("invalid amount accepted")
		}
	}
	if _, err := fx.ToUSD(100, "TOKEN"); err == nil {
		t.Fatal("token treated as fiat")
	}
	if source, date := fx.Status(); source != "frankfurter" || date != "2026-10-02" {
		t.Fatalf("status %s %s", source, date)
	}
	offline := NewCurrencyConverter(getterFunc(func(context.Context, string, map[string]string, any) error { return errors.New("offline") }), "", nil)
	if offline.Refresh(context.Background()) == nil {
		t.Fatal("expected refresh failure")
	}
	job := models.JobListing{CompensationAmount: 1600000, Currency: "IDR"}
	offline.Normalize(&job)
	if job.CompensationUSD != 100 || job.ForexSource != "offline" {
		t.Fatalf("offline conversion %+v", job)
	}
	job.Currency = "TOKEN"
	offline.Normalize(&job)
	if job.CompensationUSD != 0 || job.ForexSource != "unsupported" {
		t.Fatal("unsupported conversion not cleared")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(offline.Refresh(ctx), context.Canceled) {
		t.Fatal("cancellation not honored")
	}
}

func TestCurrencyRejectsInvalidPayload(t *testing.T) {
	for _, payload := range []string{`{"base":"EUR","date":"2026-10-02","rates":{"EUR":0.8}}`, `{"base":"USD","date":"invalid","rates":{"EUR":0.8}}`, `{"base":"USD","date":"2026-10-02","rates":{"EUR":-1,"JPY":150,"IDR":16000}}`, `{"base":"USD","date":"2026-10-02","rates":{"EUR":0.8,"JPY":150}}`} {
		fx := NewCurrencyConverter(getterFunc(func(_ context.Context, _ string, _ map[string]string, v any) error {
			return json.Unmarshal([]byte(payload), v)
		}), "", nil)
		if err := fx.Refresh(context.Background()); err == nil {
			t.Fatalf("accepted %s", payload)
		}
		if source, _ := fx.Status(); source != "offline" {
			t.Fatal("invalid rates replaced offline rates")
		}
	}
}

func TestCurrencyStaleRatesRemainLabeledAndErrorsRemainVisible(t *testing.T) {
	now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	failed := false
	calls := 0
	fx := NewCurrencyConverter(getterFunc(func(_ context.Context, _ string, _ map[string]string, v any) error {
		calls++
		if failed {
			return errors.New("offline")
		}
		return json.Unmarshal([]byte(`{"base":"USD","date":"2026-10-02","rates":{"EUR":0.8,"JPY":150,"IDR":16000}}`), v)
	}), "", func() time.Time { return now })
	if err := fx.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(31 * time.Minute)
	failed = true
	if err := fx.Refresh(context.Background()); err == nil {
		t.Fatal("expected refresh failure")
	}
	if source, _ := fx.Status(); source != "frankfurter-stale" {
		t.Fatal("stale live rate not labeled")
	}
	if value, err := fx.ToUSD(80, "EUR"); err != nil || value != 100 {
		t.Fatal("last known rate lost")
	}
	if err := fx.Refresh(context.Background()); err == nil || calls != 2 {
		t.Fatal("failure backoff lost warning or repeated request")
	}
}

func TestScoreBoundsRiskAndCompensationPeriods(t *testing.T) {
	now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	p := Profile{Skills: []string{"go", "sql"}, Text: "Go SQL", TargetHourlyUSD: 50}
	j := models.JobListing{Title: "Go SQL", SkillsRequired: []string{"go", "sql"}, CreatedAt: now, RiskStatus: models.RiskGreen, GhostingRate: 0, CompensationUSD: 104000, CompensationPeriod: "year"}
	strong := Score(j, p, now)
	if strong.WinRate != 98 || strong.SkillsMatch != 1 {
		t.Fatalf("strong score %+v", strong)
	}
	j.RiskStatus = models.RiskRed
	risky := Score(j, p, now)
	if risky.WinRate >= strong.WinRate {
		t.Fatal("risk should lower score")
	}
	j.RiskStatus = models.RiskGreen
	j.CompensationUSD = 50
	j.CompensationPeriod = "hour"
	if got := Score(j, p, now); got.WinRate != strong.WinRate {
		t.Fatal("annual and hourly pay not normalized consistently")
	}
	j.CompensationPeriod = "task"
	if _, ok := HourlyUSD(j); ok {
		t.Fatal("task annualized")
	}
	for _, ghost := range []float64{-1, 0, 1, 2, math.NaN()} {
		j.GhostingRate = ghost
		got := Score(j, p, now)
		if got.WinRate < 5 || got.WinRate > 98 || math.IsNaN(got.SkillsMatch) {
			t.Fatalf("invalid score %+v", got)
		}
	}
	weak := Score(models.JobListing{RiskStatus: models.RiskRed, CreatedAt: now.Add(-365 * 24 * time.Hour)}, Profile{}, now)
	if weak.WinRate >= strong.WinRate || weak.WinRate < 5 {
		t.Fatal("invalid weak score")
	}
}

func TestMarketPulseWindowsDedupMocksAndAging(t *testing.T) {
	now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	pulse := NewMarketPulse(func() time.Time { return now })
	job := func(id string, days int) models.JobListing {
		return models.JobListing{ID: id, Platform: "test", Category: models.CategoryIT, CreatedAt: now.Add(time.Duration(-days) * 24 * time.Hour), CompensationUSD: 104000, CompensationPeriod: "year", SkillsRequired: []string{"golang", "SQL", "sql"}}
	}
	jobs := []models.JobListing{job("old", 8), job("current1", 1), job("current2", 2), job("expired", 15)}
	mock := job("mock", 1)
	mock.IsMock = true
	jobs = append(jobs, mock)
	pulse.Observe(jobs)
	pulse.Observe(jobs)
	report := pulse.Report("IT")
	if report.DemandGrowth != 100 || report.Sentiment != "BULLISH" || report.AverageRate != "USD 50.00/hour" || len(report.TopSkills) != 2 || report.TopSkills[0] != "go" {
		t.Fatalf("report %+v", report)
	}
	if len(pulse.Snapshot()) != 3 {
		t.Fatal("duplicates, mocks or expired jobs retained")
	}
	snapshot := pulse.Snapshot()
	snapshot[0].SkillsRequired[0] = "mutated"
	if pulse.Snapshot()[0].SkillsRequired[0] == "mutated" {
		t.Fatal("snapshot aliases memory")
	}
	if empty := pulse.Report("Web3"); empty.Sentiment != "STABLE" || empty.AverageRate != "N/A" {
		t.Fatalf("empty %+v", empty)
	}
	now = now.Add(15 * 24 * time.Hour)
	if len(pulse.Snapshot()) != 0 {
		t.Fatal("history not aged out")
	}
}

func TestMarketPulseConcurrentAccess(t *testing.T) {
	pulse := NewMarketPulse(nil)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pulse.Observe([]models.JobListing{{ID: "1", CreatedAt: time.Now().Add(-time.Hour), Category: "IT"}})
			pulse.Report("IT")
			pulse.Snapshot()
		}()
	}
	wg.Wait()
}
