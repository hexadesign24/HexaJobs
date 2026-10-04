package scraper

import (
	"context"
	"errors"
	"strings"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

const RemoteOKURL = "https://remoteok.com/api"
const RemotiveURL = "https://remotive.com/api/remote-jobs"
const JobicyURL = "https://jobicy.com/api/v2/remote-jobs"

type Remote struct {
	HTTP          *HTTP
	Platform, URL string
}

func (s *Remote) Name() string          { return s.Platform }
func (s *Remote) CacheKey(Query) string { return s.URL }
func (s *Remote) CacheDuration() time.Duration {
	return 30 * time.Minute
}
func (s *Remote) Fetch(ctx context.Context, _ Query) ([]models.JobListing, error) {
	switch s.Platform {
	case "remoteok":
		return s.remoteOK(ctx)
	case "remotive":
		return s.remotive(ctx)
	case "jobicy":
		return s.jobicy(ctx)
	default:
		return nil, errors.New("unknown remote provider")
	}
}

func (s *Remote) remoteOK(ctx context.Context) ([]models.JobListing, error) {
	var rows []struct {
		ID          identifier `json:"id"`
		Position    string     `json:"position"`
		Company     string     `json:"company"`
		Location    string     `json:"location"`
		Description string     `json:"description"`
		URL         string     `json:"url"`
		ApplyURL    string     `json:"apply_url"`
		Tags        []string   `json:"tags"`
		Date        string     `json:"date"`
		Epoch       int64      `json:"epoch"`
		Min         numeric    `json:"salary_min"`
		Max         numeric    `json:"salary_max"`
	}
	if err := s.HTTP.GetJSON(ctx, s.URL, nil, &rows); err != nil {
		return nil, err
	}
	jobs := make([]models.JobListing, 0, len(rows))
	for _, r := range rows {
		if r.ID == "" || r.Position == "" {
			continue
		} // first element is API metadata
		link := r.URL
		if link == "" {
			link = r.ApplyURL
		}
		job := baseJob(s.Name(), string(r.ID), r.Position, r.Company, r.Location, r.Description, link)
		if job.Region == "" {
			job.Region = "Worldwide"
		}
		job.SkillsRequired = ExtractSkills(r.Position + " " + r.Description + " " + strings.Join(r.Tags, " "))
		job.Category = classify(r.Position + " " + strings.Join(r.Tags, " "))
		job.CreatedAt = parseDate(r.Date)
		if job.CreatedAt.IsZero() && r.Epoch > 0 {
			job.CreatedAt = time.Unix(r.Epoch, 0).UTC()
		}
		setSalary(&job, float64(r.Min), float64(r.Max), "USD", "year")
		jobs = appendValid(jobs, job)
	}
	return jobs, nil
}

func (s *Remote) remotive(ctx context.Context) ([]models.JobListing, error) {
	var response struct {
		Jobs []struct {
			ID          identifier `json:"id"`
			Title       string     `json:"title"`
			Company     string     `json:"company_name"`
			Location    string     `json:"candidate_required_location"`
			Description string     `json:"description"`
			URL         string     `json:"url"`
			Category    string     `json:"category"`
			Date        string     `json:"publication_date"`
			Salary      string     `json:"salary"`
			Tags        []string   `json:"tags"`
		} `json:"jobs"`
	}
	if err := s.HTTP.GetJSON(ctx, s.URL, nil, &response); err != nil {
		return nil, err
	}
	if response.Jobs == nil {
		return nil, errors.New("remotive response missing jobs")
	}
	jobs := make([]models.JobListing, 0, len(response.Jobs))
	for _, r := range response.Jobs {
		job := baseJob(s.Name(), string(r.ID), r.Title, r.Company, r.Location, r.Description, r.URL)
		job.Category = classify(r.Title + " " + r.Category)
		job.SkillsRequired = ExtractSkills(r.Title + " " + r.Description + " " + strings.Join(r.Tags, " "))
		job.CreatedAt = parseDate(r.Date)
		ParseCompensation(&job, r.Salary, "year")
		jobs = appendValid(jobs, job)
	}
	return jobs, nil
}

func (s *Remote) jobicy(ctx context.Context) ([]models.JobListing, error) {
	var response struct {
		Jobs []struct {
			ID          identifier `json:"id"`
			Title       string     `json:"jobTitle"`
			Company     string     `json:"companyName"`
			Location    string     `json:"jobGeo"`
			Description string     `json:"jobDescription"`
			URL         string     `json:"url"`
			Industry    []string   `json:"jobIndustry"`
			Date        string     `json:"pubDate"`
			Min         numeric    `json:"salaryMin"`
			Max         numeric    `json:"salaryMax"`
			Currency    string     `json:"salaryCurrency"`
			Period      string     `json:"salaryPeriod"`
		} `json:"jobs"`
	}
	if err := s.HTTP.GetJSON(ctx, withQuery(s.URL, map[string]string{"count": "100"}), nil, &response); err != nil {
		return nil, err
	}
	if response.Jobs == nil {
		return nil, errors.New("jobicy response missing jobs")
	}
	jobs := make([]models.JobListing, 0, len(response.Jobs))
	for _, r := range response.Jobs {
		job := baseJob(s.Name(), string(r.ID), r.Title, r.Company, r.Location, r.Description, r.URL)
		job.Category = classify(r.Title + " " + strings.Join(r.Industry, " "))
		job.CreatedAt = parseDate(r.Date)
		period := strings.ToLower(r.Period)
		switch period {
		case "annually", "annual", "yearly":
			period = "year"
		case "monthly":
			period = "month"
		case "hourly":
			period = "hour"
		case "weekly":
			period = "week"
		case "daily":
			period = "day"
		case "hour", "day", "week", "month", "year":
		default:
			period = "unknown"
		}
		setSalary(&job, float64(r.Min), float64(r.Max), r.Currency, period)
		jobs = appendValid(jobs, job)
	}
	return jobs, nil
}
