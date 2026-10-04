package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
	"hexajobs.dev/hexajobs-cli/internal/scraper"
)

type testSource struct {
	name  string
	fetch func(context.Context, scraper.Query) ([]models.JobListing, error)
}

func (s testSource) Name() string { return s.name }
func (s testSource) Fetch(c context.Context, q scraper.Query) ([]models.JobListing, error) {
	return s.fetch(c, q)
}

func testConfig(t *testing.T) Config {
	t.Helper()
	cfg, err := DefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.CacheDir = t.TempDir()
	cfg.Skills = []string{"go", "sql"}
	cfg.CircuitFailures = 2
	return cfg
}
func testJob(id string, now time.Time) models.JobListing {
	return models.JobListing{ID: id, Title: "Go Engineer", Platform: "test", ApplyURL: "https://example.com/" + id, Category: models.CategoryIT, Region: "Worldwide", SkillsRequired: []string{"go", "sql"}, CompensationAmount: 100000, Currency: "USD", CompensationPeriod: "year", CreatedAt: now.Add(-time.Hour), RiskStatus: models.RiskYellow, GhostingRate: -1}
}

func TestRouterFailoverCacheFilteringAndPulsePersistence(t *testing.T) {
	now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	var primaryCalls, fallbackCalls atomic.Int32
	primary := testSource{"primary", func(context.Context, scraper.Query) ([]models.JobListing, error) {
		primaryCalls.Add(1)
		return nil, errors.New("unavailable")
	}}
	secondary := testSource{"secondary", func(context.Context, scraper.Query) ([]models.JobListing, error) {
		fallbackCalls.Add(1)
		j := testJob("1", now)
		return []models.JobListing{j, j}, nil
	}}
	hub := scraper.Hub{Global: []scraper.Source{primary, secondary}}
	cfg := testConfig(t)
	e, err := NewEngine(cfg, Options{Hub: &hub, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		jobs, err := e.FetchJobs(context.Background(), "go", "id", "IT")
		var partial *FetchError
		if !errors.As(err, &partial) || len(jobs) != 1 {
			t.Fatalf("jobs=%v err=%v", jobs, err)
		}
		if jobs[0].CompensationUSD != 100000 || jobs[0].WinRate < 5 || jobs[0].WinRate > 98 {
			t.Fatalf("unprocessed %+v", jobs[0])
		}
	}
	if primaryCalls.Load() != 2 || fallbackCalls.Load() != 1 {
		t.Fatalf("cache/circuit calls primary=%d fallback=%d", primaryCalls.Load(), fallbackCalls.Load())
	}
	if report := e.GetMarketPulse("IT"); report.AverageRate == "N/A" || len(report.TopSkills) != 2 {
		t.Fatalf("report %+v", report)
	}
	restarted, err := NewEngine(cfg, Options{Hub: &hub, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if restarted.GetMarketPulse("IT").AverageRate == "N/A" {
		t.Fatal("market history not restored")
	}
	jobs, _ := e.FetchJobs(context.Background(), "python", "", "IT")
	if len(jobs) != 0 {
		t.Fatal("keyword filtering failed")
	}
	if _, err := e.FetchJobs(context.Background(), "", "", "other"); err == nil {
		t.Fatal("invalid category accepted")
	}
}

func TestCircuitCooldownRecoveryRateLimitAndCancellation(t *testing.T) {
	now := time.Now()
	var calls atomic.Int32
	fail := true
	source := testSource{"source", func(context.Context, scraper.Query) ([]models.JobListing, error) {
		calls.Add(1)
		if fail {
			return nil, &scraper.StatusError{StatusCode: 429, RetryAfter: 2 * time.Minute}
		}
		return []models.JobListing{testJob("1", now)}, nil
	}}
	hub := scraper.Hub{Global: []scraper.Source{source}}
	e, err := NewEngine(testConfig(t), Options{Hub: &hub, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = e.FetchJobs(context.Background(), "", "", "IT")
	_, err = e.FetchJobs(context.Background(), "", "", "IT")
	if !errors.Is(err, ErrCircuitOpen) || calls.Load() != 1 {
		t.Fatal("rate limit not opening circuit", err)
	}
	now = now.Add(119 * time.Second)
	_, _ = e.FetchJobs(context.Background(), "", "", "IT")
	if calls.Load() != 1 {
		t.Fatal("retried before Retry-After")
	}
	now = now.Add(time.Second)
	fail = false
	jobs, err := e.FetchJobs(context.Background(), "", "", "IT")
	if err != nil || len(jobs) != 1 || calls.Load() != 2 {
		t.Fatalf("recovery jobs=%v err=%v calls=%d", jobs, err, calls.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.FetchJobs(ctx, "", "", "IT"); !errors.Is(err, context.Canceled) {
		t.Fatal("cached request ignored cancellation")
	}
}

func TestWorkerPoolBoundAndConcurrentCallers(t *testing.T) {
	var active, peak, calls atomic.Int32
	hub := scraper.Hub{}
	for i := 0; i < 8; i++ {
		id := fmt.Sprint(i)
		hub.Social = append(hub.Social, testSource{"source" + id, func(ctx context.Context, _ scraper.Query) ([]models.JobListing, error) {
			calls.Add(1)
			n := active.Add(1)
			defer active.Add(-1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			select {
			case <-time.After(5 * time.Millisecond):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return []models.JobListing{testJob(id, time.Now())}, nil
		}})
	}
	cfg := testConfig(t)
	cfg.Workers = 2
	e, err := NewEngine(cfg, Options{Hub: &hub})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			jobs, err := e.FetchJobs(context.Background(), "", "", "IT")
			if err != nil || len(jobs) != 8 {
				t.Errorf("jobs=%d err=%v", len(jobs), err)
			}
			e.GetMarketPulse("IT")
			e.PurgeExpiredCache()
		}()
	}
	wg.Wait()
	if peak.Load() > 2 || calls.Load() != 8 {
		t.Fatalf("pool/stampede peak=%d calls=%d", peak.Load(), calls.Load())
	}
}

func TestFetchCancellationDoesNotTripCircuit(t *testing.T) {
	source := testSource{"slow", func(ctx context.Context, _ scraper.Query) ([]models.JobListing, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	hub := scraper.Hub{Global: []scraper.Source{source}}
	e, err := NewEngine(testConfig(t), Options{Hub: &hub})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		_, err := e.FetchJobs(ctx, "", "", "IT")
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	}
	if !e.allowSource("slow") {
		t.Fatal("caller cancellation opened breaker")
	}
}

func TestRouterForexIntegrationAndPartialSourceResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"base":"USD","date":"2026-10-02","rates":{"EUR":0.8,"JPY":150,"IDR":16000}}`)
	}))
	defer server.Close()
	j := testJob("euro", time.Now())
	j.Currency = "EUR"
	j.CompensationAmount = 80000
	source := testSource{"partial", func(context.Context, scraper.Query) ([]models.JobListing, error) {
		return []models.JobListing{j}, errors.New("one page unavailable")
	}}
	hub := scraper.Hub{Global: []scraper.Source{source}}
	e, err := NewEngine(testConfig(t), Options{Hub: &hub, HTTPClient: server.Client(), ForexEndpoint: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := e.FetchJobs(context.Background(), "", "", "IT")
	if err == nil || len(jobs) != 1 || jobs[0].CompensationUSD != 100000 || jobs[0].ForexSource != "frankfurter" {
		t.Fatalf("%+v %v", jobs, err)
	}
}

func TestDedupPreservesDistinctHNComments(t *testing.T) {
	a := testJob("1", time.Now())
	a.ApplyURL = "https://news.ycombinator.com/item?id=1&utm_source=x"
	b := testJob("2", time.Now())
	b.ApplyURL = "https://news.ycombinator.com/item?id=2"
	c := a
	c.ID = "duplicate"
	c.ApplyURL = "https://news.ycombinator.com/item?id=1"
	if got := deduplicate([]models.JobListing{a, b, c}); len(got) != 2 {
		t.Fatalf("dedup %v", got)
	}
}

func TestFilteringUsesCountryBoundariesAndSkillAliases(t *testing.T) {
	j := testJob("1", time.Now())
	j.Region = "Australia"
	if matches(j, scraper.Query{Region: "us"}) {
		t.Fatal("US matched Australia")
	}
	j.Region = "USA"
	if !matches(j, scraper.Query{Region: "us", Keyword: "golang"}) {
		t.Fatal("country or skill alias not matched")
	}
	j.Title = "Django Engineer"
	j.SkillsRequired = []string{"python"}
	if matches(j, scraper.Query{Keyword: "go"}) {
		t.Fatal("go matched Django")
	}
	j.Region = "Indonesia"
	if !matches(j, scraper.Query{Region: "id"}) {
		t.Fatal("Indonesia code not matched")
	}
}

func TestQueuedFetchHonorsCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	source := testSource{"slow", func(context.Context, scraper.Query) ([]models.JobListing, error) {
		close(started)
		<-release
		return nil, nil
	}}
	hub := scraper.Hub{Global: []scraper.Source{source}}
	e, err := NewEngine(testConfig(t), Options{Hub: &hub})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { defer close(done); _, _ = e.FetchJobs(context.Background(), "", "", "IT") }()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = e.FetchJobs(ctx, "", "", "IT")
	close(release)
	<-done
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("queued fetch ignored deadline: %v", err)
	}
}
