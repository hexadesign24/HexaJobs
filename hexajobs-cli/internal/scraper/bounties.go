package scraper

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

const GitHubSearchURL = "https://api.github.com/search/issues"

type GitHub struct {
	HTTP       *HTTP
	URL, Token string
}

func (s *GitHub) Name() string                 { return "github" }
func (s *GitHub) CacheKey(Query) string        { return s.URL }
func (s *GitHub) CacheDuration() time.Duration { return 30 * time.Minute }

func (s *GitHub) Fetch(ctx context.Context, _ Query) ([]models.JobListing, error) {
	jobs := []models.JobListing{}
	seen := map[string]bool{}
	var failures []error
	for _, label := range []string{"bounty", "paid-task"} {
		var response struct {
			Items []struct {
				ID          identifier `json:"id"`
				Title       string     `json:"title"`
				Body        string     `json:"body"`
				URL         string     `json:"html_url"`
				State       string     `json:"state"`
				Created     string     `json:"created_at"`
				Repository  string     `json:"repository_url"`
				PullRequest jsonMarker `json:"pull_request"`
			} `json:"items"`
			Incomplete bool `json:"incomplete_results"`
		}
		headers := map[string]string{"Accept": "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28"}
		if s.Token != "" {
			headers["Authorization"] = "Bearer " + s.Token
		}
		endpoint := withQuery(s.URL, map[string]string{"q": "label:" + label + " state:open is:issue is:public", "sort": "created", "order": "desc", "per_page": "100"})
		if err := s.HTTP.GetJSON(ctx, endpoint, headers, &response); err != nil {
			failures = append(failures, err)
			continue
		}
		if response.Items == nil {
			failures = append(failures, errors.New("GitHub response missing items"))
			continue
		}
		if response.Incomplete {
			failures = append(failures, errors.New("GitHub search results incomplete"))
		}
		for _, r := range response.Items {
			if seen[string(r.ID)] || bool(r.PullRequest) || r.State != "open" {
				continue
			}
			seen[string(r.ID)] = true
			repository, _ := url.Parse(r.Repository)
			company := ""
			if repository != nil {
				company = strings.TrimPrefix(repository.Path, "/repos/")
			}
			job := baseJob(s.Name(), string(r.ID), r.Title, company, "Worldwide", r.Body, r.URL)
			job.Category = models.CategoryBounty
			job.CreatedAt = parseDate(r.Created)
			ParseCompensation(&job, r.Title+" "+r.Body, "task")
			jobs = appendValid(jobs, job)
		}
	}
	return jobs, errors.Join(failures...)
}

type jsonMarker bool

func (m *jsonMarker) UnmarshalJSON(b []byte) error { *m = string(b) != "null"; return nil }
