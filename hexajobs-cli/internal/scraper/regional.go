package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

const AdzunaURL = "https://api.adzuna.com/v1/api/jobs"

type Adzuna struct {
	HTTP                        *HTTP
	URL, AppID, AppKey, Country string
}

func (s *Adzuna) Name() string { return "adzuna" }

func (s *Adzuna) CacheKey(q Query) string {
	data, _ := json.Marshal(q)
	mode := "live"
	if strings.TrimSpace(s.AppID) == "" {
		mode = "mock"
	}
	return s.URL + ":" + s.Country + ":" + mode + ":" + string(data)
}
func (s *Adzuna) CacheDuration() time.Duration { return 30 * time.Minute }

// A fixed sample date prevents mock records from masquerading as fresh jobs.
const adzunaMock = `{"results":[{"id":"demo-go","title":"Go Backend Engineer (demo)","company":{"display_name":"Example Company"},"location":{"display_name":"Worldwide"},"description":"Go SQL Docker backend development. Demonstration data only.","redirect_url":"https://example.com/jobs/demo-go","created":"2026-01-01T00:00:00Z","salary_min":50000,"salary_max":70000,"category":{"label":"IT Jobs"}}]}`

func (s *Adzuna) Fetch(ctx context.Context, q Query) ([]models.JobListing, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var response struct {
		Results []struct {
			ID          identifier `json:"id"`
			Title       string     `json:"title"`
			Description string     `json:"description"`
			URL         string     `json:"redirect_url"`
			Created     string     `json:"created"`
			Min         numeric    `json:"salary_min"`
			Max         numeric    `json:"salary_max"`
			Company     struct {
				Name string `json:"display_name"`
			} `json:"company"`
			Location struct {
				Name string `json:"display_name"`
			} `json:"location"`
			Category struct {
				Label string `json:"label"`
			} `json:"category"`
		} `json:"results"`
	}
	mock := strings.TrimSpace(s.AppID) == ""
	country := q.Country
	if country == "" {
		country = s.Country
	}
	country = strings.ToLower(country)
	currency := map[string]string{"gb": "GBP", "us": "USD", "au": "AUD", "at": "EUR", "be": "EUR", "br": "BRL", "ca": "CAD", "ch": "CHF", "de": "EUR", "es": "EUR", "fr": "EUR", "in": "INR", "it": "EUR", "mx": "MXN", "nl": "EUR", "nz": "NZD", "pl": "PLN", "sg": "SGD", "za": "ZAR"}[country]
	if mock {
		if err := json.Unmarshal([]byte(adzunaMock), &response); err != nil {
			return nil, err
		}
		currency = "USD"
	} else {
		if s.AppKey == "" {
			return nil, errors.New("Adzuna app key missing")
		}
		if currency == "" {
			return nil, errors.New("unsupported Adzuna country")
		}
		where := q.Region
		if strings.EqualFold(where, country) {
			where = ""
		}
		endpoint := strings.TrimRight(s.URL, "/") + "/" + url.PathEscape(country) + "/search/1"
		endpoint = withQuery(endpoint, map[string]string{"app_id": s.AppID, "app_key": s.AppKey, "what": q.Keyword, "where": where, "results_per_page": "50", "content-type": "application/json"})
		if err := s.HTTP.GetJSON(ctx, endpoint, nil, &response); err != nil {
			return nil, err
		}
	}
	if response.Results == nil {
		return nil, errors.New("Adzuna response missing results")
	}
	jobs := make([]models.JobListing, 0, len(response.Results))
	for _, r := range response.Results {
		job := baseJob(s.Name(), string(r.ID), r.Title, r.Company.Name, r.Location.Name, r.Description, r.URL)
		if !mock {
			job.Region += " " + strings.ToUpper(country)
		}
		job.Category = classify(r.Title + " " + r.Category.Label)
		job.CreatedAt = parseDate(r.Created)
		job.IsMock = mock
		if mock {
			job.Platform = "adzuna-mock"
			job.ID = "adzuna-mock:" + string(r.ID)
		}
		setSalary(&job, float64(r.Min), float64(r.Max), currency, "year")
		jobs = appendValid(jobs, job)
	}
	return jobs, nil
}
