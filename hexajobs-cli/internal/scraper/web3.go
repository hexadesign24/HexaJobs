package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

type Web3 struct {
	HTTP *HTTP
	URL  string
}

func (s *Web3) Name() string                 { return "web3" }
func (s *Web3) CacheKey(Query) string        { return s.URL }
func (s *Web3) CacheDuration() time.Duration { return 30 * time.Minute }

const web3Mock = `{"tasks":[{"id":"demo-testnet","title":"Testnet QA (demo)","project":"Example Protocol","kind":"testnet","url":"https://example.com/testnet","skills":["ethereum","qa"],"reward":{"amount":0,"currency":"TOKEN","guaranteed":false},"audit":{"status":"unknown"},"created_at":"2026-01-01T00:00:00Z"},{"id":"demo-bounty","title":"Solidity test bounty (demo)","project":"Example Protocol","kind":"bounty","url":"https://example.com/bounty","skills":["solidity"],"reward":{"amount":250,"currency":"USD","guaranteed":true},"audit":{"status":"passed"},"created_at":"2026-01-01T00:00:00Z"}]}`

func (s *Web3) Fetch(ctx context.Context, _ Query) ([]models.JobListing, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var response struct {
		Tasks []struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Project     string   `json:"project"`
			Kind        string   `json:"kind"`
			URL         string   `json:"url"`
			Description string   `json:"description"`
			Region      string   `json:"region"`
			Skills      []string `json:"skills"`
			Created     string   `json:"created_at"`
			Reward      struct {
				Amount     float64 `json:"amount"`
				Currency   string  `json:"currency"`
				Guaranteed bool    `json:"guaranteed"`
			} `json:"reward"`
			Audit struct {
				Status string `json:"status"`
			} `json:"audit"`
			RequiresDeposit bool `json:"requires_deposit"`
		} `json:"tasks"`
	}
	mock := s.URL == ""
	if mock {
		if err := json.Unmarshal([]byte(web3Mock), &response); err != nil {
			return nil, err
		}
	} else if err := s.HTTP.GetJSON(ctx, s.URL, nil, &response); err != nil {
		return nil, err
	}
	if response.Tasks == nil {
		return nil, errors.New("Web3 response missing tasks")
	}
	jobs := make([]models.JobListing, 0, len(response.Tasks))
	for _, r := range response.Tasks {
		job := baseJob(s.Name(), r.ID, r.Title, r.Project, r.Region, r.Description, r.URL)
		job.IsMock = mock
		if mock {
			job.Platform = "web3-mock"
			job.ID = "web3-mock:" + r.ID
		}
		if job.Region == "" {
			job.Region = "Worldwide"
		}
		job.Category = models.CategoryWeb3
		if r.Kind == "bounty" {
			job.Category = models.CategoryBounty
		}
		job.CreatedAt = parseDate(r.Created)
		job.SkillsRequired = r.Skills
		job.AuditStatus = strings.ToLower(r.Audit.Status)
		switch job.AuditStatus {
		case "passed":
			if r.Reward.Guaranteed {
				job.RiskStatus = models.RiskGreen
			}
		case "failed":
			job.RiskStatus = models.RiskRed
		default:
			job.AuditStatus = "unknown"
		}
		if r.RequiresDeposit {
			job.RiskStatus = models.RiskRed
		}
		job.CompensationPeriod = "task"
		if r.Reward.Guaranteed {
			setSalary(&job, r.Reward.Amount, r.Reward.Amount, r.Reward.Currency, "task")
		} else {
			job.CompensationRaw = "Unconfirmed reward (" + r.Reward.Currency + ")"
			if job.RiskStatus != models.RiskRed {
				job.RiskStatus = models.RiskYellow
			}
		}
		jobs = appendValid(jobs, job)
	}
	return jobs, nil
}
