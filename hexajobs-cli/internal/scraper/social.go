package scraper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

const HackerNewsURL = "https://hacker-news.firebaseio.com/v0"

type HackerNews struct {
	HTTP                 *HTTP
	URL                  string
	ThreadID             int64
	MaxComments, Workers int
}

func (s *HackerNews) Name() string { return "hackernews" }
func (s *HackerNews) CacheKey(Query) string {
	return fmt.Sprintf("%s:%d:%d", s.URL, s.ThreadID, s.MaxComments)
}
func (s *HackerNews) CacheDuration() time.Duration { return 30 * time.Minute }

type hnItem struct {
	ID      int64   `json:"id"`
	Type    string  `json:"type"`
	Title   string  `json:"title"`
	Text    string  `json:"text"`
	By      string  `json:"by"`
	Time    int64   `json:"time"`
	Kids    []int64 `json:"kids"`
	Parent  int64   `json:"parent"`
	Deleted bool    `json:"deleted"`
	Dead    bool    `json:"dead"`
}

func (s *HackerNews) items(ctx context.Context, ids []int64) ([]hnItem, error) {
	type result struct {
		item hnItem
		err  error
	}
	queue := make(chan int64, len(ids))
	results := make(chan result, len(ids))
	for _, id := range ids {
		queue <- id
	}
	close(queue)
	workers := s.Workers
	if workers < 1 {
		workers = 4
	}
	if workers > len(ids) {
		workers = len(ids)
	}
	if workers > 32 {
		workers = 32
	}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range queue {
				if ctx.Err() != nil {
					return
				}
				var item hnItem
				err := s.HTTP.GetJSON(ctx, fmt.Sprintf("%s/item/%d.json", strings.TrimRight(s.URL, "/"), id), nil, &item)
				results <- result{item, err}
			}
		}()
	}
	wg.Wait()
	close(results)
	var items []hnItem
	var failures []error
	for r := range results {
		if r.err != nil {
			failures = append(failures, r.err)
		} else {
			items = append(items, r.item)
		}
	}
	if ctx.Err() != nil {
		failures = append(failures, ctx.Err())
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, errors.Join(failures...)
}

func (s *HackerNews) Fetch(ctx context.Context, _ Query) ([]models.JobListing, error) {
	var thread hnItem
	if s.ThreadID > 0 {
		items, err := s.items(ctx, []int64{s.ThreadID})
		if err != nil {
			return nil, err
		}
		thread = items[0]
	} else {
		var user struct {
			Submitted []int64 `json:"submitted"`
		}
		if err := s.HTTP.GetJSON(ctx, strings.TrimRight(s.URL, "/")+"/user/whoishiring.json", nil, &user); err != nil {
			return nil, err
		}
		if len(user.Submitted) > 30 {
			user.Submitted = user.Submitted[:30]
		}
		items, err := s.items(ctx, user.Submitted)
		for _, item := range items {
			if isHiringThread(item) && item.Time > thread.Time {
				thread = item
			}
		}
		if thread.ID == 0 {
			if err != nil {
				return nil, err
			}
			return nil, errors.New("HN hiring thread not found")
		}
	}
	if !isHiringThread(thread) {
		return nil, errors.New("HN item is not an active Who is hiring thread")
	}
	limit := s.MaxComments
	if limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	ids := thread.Kids
	if len(ids) > limit {
		ids = ids[:limit]
	}
	items, err := s.items(ctx, ids)
	jobs := make([]models.JobListing, 0, len(items))
	for _, item := range items {
		if item.Deleted || item.Dead || item.Type != "comment" || item.Parent != thread.ID {
			continue
		}
		text := PlainText(item.Text)
		if text == "" {
			continue
		}
		header := strings.SplitN(item.Text, "<p>", 2)[0]
		header = PlainText(header)
		chars := []rune(header)
		if len(chars) > 180 {
			header = string(chars[:180])
		}
		company := strings.TrimSpace(strings.SplitN(header, "|", 2)[0])
		region := "Unspecified"
		if containsTerm(strings.ToLower(text), "remote") {
			region = "Remote (restrictions unspecified)"
		}
		job := baseJob(s.Name(), fmt.Sprint(item.ID), header, company, region, text, fmt.Sprintf("https://news.ycombinator.com/item?id=%d", item.ID))
		if item.Time > 0 {
			job.CreatedAt = time.Unix(item.Time, 0).UTC()
		}
		// Only parse the hiring header: company funding in the body is not salary.
		ParseCompensation(&job, header, "unknown")
		jobs = appendValid(jobs, job)
	}
	return jobs, err
}

func isHiringThread(item hnItem) bool {
	return item.ID > 0 && item.Type == "story" && !item.Deleted && !item.Dead && strings.HasPrefix(strings.ToLower(PlainText(item.Title)), "ask hn: who is hiring?")
}

// Forum ingests an explicitly configured public RSS 2.0 job/gig feed.
type Forum struct {
	HTTP *HTTP
	URL  string
}

func (s *Forum) Name() string {
	hash := sha256.Sum256([]byte(s.URL))
	return "forum-" + hex.EncodeToString(hash[:6])
}
func (s *Forum) CacheKey(Query) string        { return s.URL }
func (s *Forum) CacheDuration() time.Duration { return 30 * time.Minute }
func (s *Forum) Fetch(ctx context.Context, _ Query) ([]models.JobListing, error) {
	data, err := s.HTTP.Get(ctx, s.URL, map[string]string{"Accept": "application/rss+xml, application/xml"})
	if err != nil {
		return nil, err
	}
	var feed struct {
		XMLName xml.Name `xml:"rss"`
		Channel struct {
			Title string `xml:"title"`
			Items []struct {
				GUID        string `xml:"guid"`
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
				Date        string `xml:"pubDate"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, errors.New("invalid public RSS feed")
	}
	jobs := []models.JobListing{}
	for i, r := range feed.Channel.Items {
		if i >= 500 {
			break
		}
		id := r.GUID
		if id == "" {
			sum := sha256.Sum256([]byte(r.Link))
			id = hex.EncodeToString(sum[:])
		}
		job := baseJob(s.Name(), id, r.Title, feed.Channel.Title, "Unspecified", r.Description, r.Link)
		job.CreatedAt = parseDate(r.Date)
		ParseCompensation(&job, r.Title, "unknown")
		jobs = appendValid(jobs, job)
	}
	return jobs, nil
}
