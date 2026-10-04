// Package scraper ingests public feeds and returns normalized records, never UI output.
package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

const UserAgent = "hexajobs-cli/1.0"
const MaxBodyBytes = 16 << 20

type Query struct{ Keyword, Region, Category, Country string }

type Source interface {
	Name() string
	Fetch(context.Context, Query) ([]models.JobListing, error)
}

// CachePolicy may override the query cache key and increase TTL to respect feed limits.
type CachePolicy interface {
	CacheKey(Query) string
	CacheDuration() time.Duration
}

type HTTP struct {
	client *http.Client
	slots  chan struct{}
}

func NewHTTP(client *http.Client, workers int, timeout time.Duration) *HTTP {
	if workers < 1 {
		workers = 1
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	copyClient := http.Client{}
	if client != nil {
		copyClient = *client
	}
	if copyClient.Timeout <= 0 || copyClient.Timeout > timeout {
		copyClient.Timeout = timeout
	}
	// Do not forward query credentials or authorization headers through redirects
	// to another host. Public HTTPS API migrations remain supported.
	previous := copyClient.CheckRedirect
	copyClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		if len(via) > 0 {
			original := via[0]
			if original.URL.Scheme == "https" && req.URL.Scheme != "https" {
				return http.ErrUseLastResponse
			}
			secret := original.Header.Get("Authorization") != "" || original.URL.Query().Has("app_key") || original.URL.Query().Has("app_id")
			if secret && req.URL.Host != original.URL.Host {
				return http.ErrUseLastResponse
			}
		}
		if previous != nil {
			return previous(req, via)
		}
		return nil
	}
	return &HTTP{client: &copyClient, slots: make(chan struct{}, workers)}
}

type StatusError struct {
	StatusCode int
	RetryAfter time.Duration
}

func (e *StatusError) Error() string { return fmt.Sprintf("HTTP status %d", e.StatusCode) }

func (h *HTTP) Get(ctx context.Context, endpoint string, headers map[string]string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	u, err := url.Parse(endpoint)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
		return nil, errors.New("invalid feed URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New("invalid HTTP request")
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// url.Error includes query parameters (Adzuna credentials); never expose it.
		return nil, errors.New("feed HTTP transport failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retry := time.Duration(0)
		if seconds, parseErr := time.ParseDuration(resp.Header.Get("Retry-After") + "s"); parseErr == nil {
			retry = seconds
		} else if until, parseErr := http.ParseTime(resp.Header.Get("Retry-After")); parseErr == nil {
			retry = time.Until(until)
		}
		return nil, &StatusError{StatusCode: resp.StatusCode, RetryAfter: retry}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxBodyBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("failed reading feed response")
	}
	if len(data) > MaxBodyBytes {
		return nil, errors.New("feed response exceeds size limit")
	}
	return data, nil
}

func (h *HTTP) GetJSON(ctx context.Context, endpoint string, headers map[string]string, target any) error {
	data, err := h.Get(ctx, endpoint, headers)
	if err != nil {
		return err
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return errors.New("null feed response")
	}
	if err = json.Unmarshal(data, target); err != nil {
		return errors.New("invalid feed JSON")
	}
	return nil
}

type Hub struct {
	Global                           []Source
	Regional, Social, Bounties, Web3 []Source
}

// Routes returns ordered failover chains; separate chains run concurrently.
func (h Hub) Routes(q Query) [][]Source {
	var routes [][]Source
	if q.Category == "" || q.Category == models.CategoryIT || q.Category == models.CategoryNonIT || q.Category == models.CategoryWeb3 {
		if len(h.Global) > 0 {
			routes = append(routes, h.Global)
		}
		for _, source := range h.Social {
			routes = append(routes, []Source{source})
		}
	}
	if q.Category == "" || q.Category == models.CategoryIT || q.Category == models.CategoryNonIT {
		for _, source := range h.Regional {
			routes = append(routes, []Source{source})
		}
	}
	if q.Category == "" || q.Category == models.CategoryBounty {
		for _, source := range h.Bounties {
			routes = append(routes, []Source{source})
		}
	}
	if q.Category == "" || q.Category == models.CategoryWeb3 || q.Category == models.CategoryBounty {
		for _, source := range h.Web3 {
			routes = append(routes, []Source{source})
		}
	}
	return routes
}

func withQuery(endpoint string, params map[string]string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	values := u.Query()
	for key, value := range params {
		if value != "" {
			values.Set(key, value)
		}
	}
	u.RawQuery = values.Encode()
	return u.String()
}

func validURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil
}

func baseJob(platform, id, title, company, region, description, applyURL string) models.JobListing {
	return models.JobListing{ID: platform + ":" + id, Platform: platform, Title: PlainText(title), Company: PlainText(company), Region: PlainText(region), Description: PlainText(description), ApplyURL: applyURL, Category: classify(title + " " + description), SkillsRequired: ExtractSkills(title + " " + description), RiskStatus: models.RiskYellow, GhostingRate: -1, CompensationPeriod: "unknown"}
}

func appendValid(jobs []models.JobListing, job models.JobListing) []models.JobListing {
	if job.Title == "" || strings.HasSuffix(job.ID, ":") || !validURL(job.ApplyURL) {
		return jobs
	}
	return append(jobs, job)
}
