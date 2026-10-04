package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/engine"
	"hexajobs.dev/hexajobs-cli/internal/models"
	"hexajobs.dev/hexajobs-cli/internal/scraper"
)

var ErrCircuitOpen = errors.New("source circuit breaker open")

// FetchError accompanies partial results. Callers should retain jobs even when
// err != nil and inspect Failures for unavailable sources or persistence errors.
type FetchError struct{ Failures map[string]error }

func (e *FetchError) Error() string {
	keys := make([]string, 0, len(e.Failures))
	for key := range e.Failures {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+": "+e.Failures[key].Error())
	}
	return strings.Join(parts, "; ")
}
func (e *FetchError) Unwrap() []error {
	out := make([]error, 0, len(e.Failures))
	for _, err := range e.Failures {
		out = append(out, err)
	}
	return out
}

type breaker struct {
	failures int
	until    time.Time
	probing  bool
}
type Engine struct {
	cfg      Config
	cache    *Cache
	hub      scraper.Hub
	fx       *engine.CurrencyConverter
	pulse    *engine.MarketPulse
	now      func() time.Time
	gate     chan struct{}
	mu       sync.Mutex
	breakers map[string]*breaker
}

var _ models.EngineContract = (*Engine)(nil)

// Options enables deterministic tests and alternative transports/feed adapters.
// Config and Hub are copied; sources and HTTP transports must be concurrency safe.
type Options struct {
	HTTPClient    *http.Client
	Hub           *scraper.Hub
	ForexEndpoint string
	Now           func() time.Time
}

func NewEngine(cfg Config, opts Options) (*Engine, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cache, err := NewCache(cfg.CacheDir)
	if err != nil {
		return nil, err
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	cache.now = now
	h := scraper.NewHTTP(opts.HTTPClient, cfg.Workers, cfg.HTTPTimeout())
	hub := scraper.Hub{
		Global:   []scraper.Source{&scraper.Remote{HTTP: h, Platform: "remoteok", URL: scraper.RemoteOKURL}, &scraper.Remote{HTTP: h, Platform: "remotive", URL: scraper.RemotiveURL}, &scraper.Remote{HTTP: h, Platform: "jobicy", URL: scraper.JobicyURL}},
		Regional: []scraper.Source{&scraper.Adzuna{HTTP: h, URL: scraper.AdzunaURL, AppID: cfg.AdzunaAppID, AppKey: cfg.AdzunaAppKey, Country: cfg.AdzunaCountry}},
		Social:   []scraper.Source{&scraper.HackerNews{HTTP: h, URL: scraper.HackerNewsURL, ThreadID: cfg.HNThreadID, MaxComments: cfg.HNMaxComments, Workers: cfg.Workers}},
		Bounties: []scraper.Source{&scraper.GitHub{HTTP: h, URL: scraper.GitHubSearchURL, Token: cfg.GitHubToken}},
		Web3:     []scraper.Source{&scraper.Web3{HTTP: h, URL: cfg.Web3FeedURL}},
	}
	for _, endpoint := range cfg.ForumFeedURLs {
		hub.Social = append(hub.Social, &scraper.Forum{HTTP: h, URL: endpoint})
	}
	if opts.Hub != nil {
		hub = *opts.Hub
	}
	hub.Global = append([]scraper.Source(nil), hub.Global...)
	hub.Regional = append([]scraper.Source(nil), hub.Regional...)
	hub.Social = append([]scraper.Source(nil), hub.Social...)
	hub.Bounties = append([]scraper.Source(nil), hub.Bounties...)
	hub.Web3 = append([]scraper.Source(nil), hub.Web3...)
	cfg.Skills = append([]string(nil), cfg.Skills...)
	e := &Engine{cfg: cfg, cache: cache, hub: hub, fx: engine.NewCurrencyConverter(h, opts.ForexEndpoint, now), pulse: engine.NewMarketPulse(now), now: now, gate: make(chan struct{}, 1), breakers: make(map[string]*breaker)}
	e.PurgeExpiredCache()
	if history, err := cache.GetWithTTL("market-history:v1", CacheRetention); err == nil {
		e.pulse.Observe(history)
	}
	return e, nil
}

func canonicalCategory(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return "", nil
	case "it":
		return models.CategoryIT, nil
	case "non-it":
		return models.CategoryNonIT, nil
	case "web3":
		return models.CategoryWeb3, nil
	case "bounty":
		return models.CategoryBounty, nil
	default:
		return "", fmt.Errorf("invalid category %q", value)
	}
}

func (e *Engine) FetchJobs(ctx context.Context, keyword, region, category string) ([]models.JobListing, error) {
	if ctx == nil {
		return nil, errors.New("nil context")
	}
	cat, err := canonicalCategory(category)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Serializing batches coalesces repeated source requests through the cache,
	// bounds memory, and keeps circuit half-open probes single-flight.
	select {
	case e.gate <- struct{}{}:
		defer func() { <-e.gate }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.PurgeExpiredCache()
	q := scraper.Query{Keyword: strings.ToLower(strings.TrimSpace(keyword)), Region: strings.ToLower(strings.TrimSpace(region)), Category: cat, Country: e.cfg.AdzunaCountry}
	if len(q.Region) == 2 && strings.Trim(q.Region, "abcdefghijklmnopqrstuvwxyz") == "" {
		q.Country = q.Region
	}
	routes := e.hub.Routes(q)
	type result struct {
		index    int
		jobs     []models.JobListing
		failures map[string]error
	}
	queue := make(chan int, len(routes))
	results := make(chan result, len(routes))
	for i := range routes {
		queue <- i
	}
	close(queue)
	var wg sync.WaitGroup
	workers := min(e.cfg.Workers, len(routes))
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range queue {
				if ctx.Err() != nil {
					return
				}
				jobs, failures := e.fetchRoute(ctx, q, routes[index])
				results <- result{index, jobs, failures}
			}
		}()
	}
	wg.Wait()
	close(results)
	ordered := make([]result, len(routes))
	for r := range results {
		ordered[r.index] = r
	}
	failures := make(map[string]error)
	all := []models.JobListing{}
	for _, r := range ordered {
		all = append(all, r.jobs...)
		for name, err := range r.failures {
			failures[name] = err
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	needsFX := false
	for _, job := range all {
		if job.CompensationAmount > 0 && job.Currency != "USD" {
			needsFX = true
			break
		}
	}
	if needsFX {
		if err := e.fx.Refresh(ctx); err != nil {
			failures["forex"] = err
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	all = deduplicate(all)
	for i := range all {
		e.fx.Normalize(&all[i])
	}
	e.pulse.Observe(all)
	if err := e.cache.Put("market-history:v1", e.pulse.Snapshot()); err != nil {
		failures["market-cache"] = err
	}
	profile := engine.Profile{Skills: e.cfg.Skills, Text: e.cfg.ProfileText, TargetHourlyUSD: e.cfg.TargetHourlyUSD}
	jobs := make([]models.JobListing, 0, len(all))
	for _, job := range all {
		if matches(job, q) {
			jobs = append(jobs, engine.Score(job, profile, e.now()))
		}
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		a, b := jobs[i], jobs[j]
		if a.IsMock != b.IsMock {
			return !a.IsMock
		}
		if a.WinRate != b.WinRate {
			return a.WinRate > b.WinRate
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		return a.ID < b.ID
	})
	if len(failures) > 0 {
		return jobs, &FetchError{Failures: failures}
	}
	return jobs, nil
}

func (e *Engine) fetchRoute(ctx context.Context, q scraper.Query, sources []scraper.Source) ([]models.JobListing, map[string]error) {
	failures := make(map[string]error)
	jobs := []models.JobListing{}
	for _, source := range sources {
		if ctx.Err() != nil {
			return jobs, failures
		}
		data, warning, err := e.fetchSource(ctx, q, source)
		jobs = append(jobs, data...)
		if warning != nil {
			failures[source.Name()+"-cache"] = warning
		}
		if err != nil {
			failures[source.Name()] = err
			continue
		}
		if len(data) > 0 {
			return jobs, failures
		}
	}
	return jobs, failures
}

func (e *Engine) fetchSource(ctx context.Context, q scraper.Query, source scraper.Source) ([]models.JobListing, error, error) {
	keyBytes, _ := json.Marshal(q)
	key := string(keyBytes)
	ttl := CacheTTL
	if policy, ok := source.(scraper.CachePolicy); ok {
		key = policy.CacheKey(q)
		ttl = max(ttl, policy.CacheDuration())
	}
	key = "source:v1:" + source.Name() + ":" + key
	if jobs, err := e.cache.GetWithTTL(key, ttl); err == nil {
		return jobs, nil, nil
	}
	if !e.allowSource(source.Name()) {
		return nil, nil, ErrCircuitOpen
	}
	jobs, err := source.Fetch(ctx, q)
	e.recordResult(source.Name(), err, ctx.Err() != nil)
	if err != nil {
		return jobs, nil, err
	}
	return jobs, e.cache.Put(key, jobs), nil
}

func (e *Engine) allowSource(name string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	b := e.breakers[name]
	if b == nil {
		b = &breaker{}
		e.breakers[name] = b
	}
	if b.until.IsZero() {
		return true
	}
	if e.now().Before(b.until) || b.probing {
		return false
	}
	b.probing = true
	return true
}

func (e *Engine) recordResult(name string, err error, cancelled bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	b := e.breakers[name]
	b.probing = false
	if cancelled {
		return
	}
	if err == nil {
		b.failures = 0
		b.until = time.Time{}
		return
	}
	b.failures++
	cooldown := time.Duration(e.cfg.CircuitCooldownSeconds) * time.Second
	var status *scraper.StatusError
	limited := errors.As(err, &status) && (status.StatusCode == http.StatusTooManyRequests || status.StatusCode == http.StatusForbidden)
	if limited && status.RetryAfter > cooldown {
		cooldown = min(status.RetryAfter, 24*time.Hour)
	}
	if limited || b.failures >= e.cfg.CircuitFailures || !b.until.IsZero() {
		b.until = e.now().Add(cooldown)
	}
}

func matches(job models.JobListing, q scraper.Query) bool {
	if q.Category != "" && job.Category != q.Category {
		return false
	}
	text := job.Title + " " + job.Company + " " + job.Description + " " + strings.Join(job.SkillsRequired, " ")
	words := make(map[string]bool)
	for _, word := range engine.Tokens(text) {
		words[engine.NormalizeSkill(strings.Trim(word, "."))] = true
	}
	for _, word := range engine.Tokens(q.Keyword) {
		if !words[engine.NormalizeSkill(strings.Trim(word, "."))] {
			return false
		}
	}
	region := strings.ToLower(job.Region)
	if q.Region == "" || q.Region == "all" || region == "worldwide" || region == "anywhere" || region == "global" {
		return true
	}
	if len(q.Region) == 2 {
		aliases := map[string][]string{"us": {"us", "usa", "united states"}, "gb": {"gb", "uk", "united kingdom"}, "id": {"id", "indonesia"}, "au": {"au", "australia"}, "ca": {"ca", "canada"}, "de": {"de", "germany"}, "fr": {"fr", "france"}, "jp": {"jp", "japan"}, "sg": {"sg", "singapore"}, "in": {"in", "india"}}
		candidates := aliases[q.Region]
		if len(candidates) == 0 {
			candidates = []string{q.Region}
		}
		location := " " + strings.Join(engine.Tokens(region), " ") + " "
		for _, alias := range candidates {
			if strings.Contains(location, " "+alias+" ") {
				return true
			}
		}
		return false
	}
	return strings.Contains(region, q.Region)
}

func deduplicate(jobs []models.JobListing) []models.JobListing {
	ids, urls := map[string]bool{}, map[string]bool{}
	out := make([]models.JobListing, 0, len(jobs))
	for _, job := range jobs {
		if job.ID == "" || job.Title == "" {
			continue
		}
		key := job.Platform + ":" + job.ID
		u, err := url.Parse(job.ApplyURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			continue
		}
		u.Fragment = ""
		u.Host = strings.ToLower(u.Host)
		query := u.Query()
		for param := range query {
			if strings.HasPrefix(strings.ToLower(param), "utm_") {
				query.Del(param)
			}
		}
		u.RawQuery = query.Encode()
		link := u.String()
		if ids[key] || urls[link] {
			continue
		}
		ids[key] = true
		urls[link] = true
		out = append(out, job)
	}
	return out
}

func (e *Engine) GetMarketPulse(sector string) models.MarketReport { return e.pulse.Report(sector) }
func (e *Engine) PurgeExpiredCache() int                           { return e.cache.PurgeExpired() }
