package scraper

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

func TestParseCompensation(t *testing.T) {
	for _, tc := range []struct {
		raw, currency, period string
		want                  float64
	}{
		{"$80-100k/year", "USD", "year", 90000}, {"EUR 40 - 60 per hour", "EUR", "hour", 50}, {"Rp 15.000.000 - 25.000.000/month", "IDR", "month", 20000000},
		{"€50,50 per hour", "EUR", "hour", 50.5}, {"120000 USD annually", "USD", "year", 120000}, {"JPY 5,000,000/year", "JPY", "year", 5000000},
		{"Competitive", "", "unknown", 0}, {"1000 USDT", "", "unknown", 0}, {"AUD 100000/year", "AUD", "year", 100000},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			j := models.JobListing{CompensationPeriod: "unknown"}
			ParseCompensation(&j, tc.raw, "unknown")
			if math.Abs(j.CompensationAmount-tc.want) > 1e-5 || j.Currency != tc.currency || j.CompensationPeriod != tc.period {
				t.Fatalf("got %+v", j)
			}
			if j.CompensationRaw != tc.raw {
				t.Fatal("original pay lost")
			}
		})
	}
}

func TestPlainTextAndSkills(t *testing.T) {
	text := PlainText(`<p>Go &amp; SQL</p><script>bad()</script>` + string(rune(27)))
	if text != "Go & SQL" {
		t.Fatalf("plain text %q", text)
	}
	skills := ExtractSkills("Django, javascript and C++ with Golang. Not Java or C#?")
	if !strings.Contains(strings.Join(skills, ","), "go") {
		t.Fatal(skills)
	}
	if got := ExtractSkills("Django javascript"); strings.Contains(strings.Join(got, ","), "go") {
		t.Fatalf("substring skill match %v", got)
	}
}

func TestRemoteAdapters(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		want          float64
		period        string
	}{
		{"remoteok", `[{"legal":"metadata"},{"id":"1","position":"Go Engineer","company":"Acme","url":"https://remoteok.com/jobs/1","apply_url":"https://employer.test/apply","salary_min":100000,"salary_max":120000,"tags":["golang"],"date":"2026-10-01T12:00:00Z"}]`, 110000, "year"},
		{"remotive", `{"jobs":[{"id":2,"title":"Go Developer","company_name":"Acme","candidate_required_location":"Worldwide","url":"https://remotive.com/jobs/2","salary":"EUR 40-60 per hour","publication_date":"2026-10-01T12:00:00"}]}`, 50, "hour"},
		{"jobicy", `{"jobs":[{"id":3,"jobTitle":"Go Developer","companyName":"Acme","jobGeo":"USA","url":"https://jobicy.com/jobs/3","salaryMin":"90000","salaryMax":110000,"salaryCurrency":"USD","salaryPeriod":"yearly","jobIndustry":["Engineering"],"pubDate":"2026-10-01 12:00:00"}]}`, 100000, "year"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("User-Agent") != UserAgent {
					t.Error("missing custom user agent")
				}
				fmt.Fprint(w, tc.payload)
			}))
			defer server.Close()
			s := &Remote{HTTP: NewHTTP(server.Client(), 2, time.Second), Platform: tc.name, URL: server.URL}
			jobs, err := s.Fetch(context.Background(), Query{})
			if err != nil || len(jobs) != 1 {
				t.Fatalf("%v %v", jobs, err)
			}
			j := jobs[0]
			if j.CompensationAmount != tc.want || j.CompensationPeriod != tc.period || j.CreatedAt.IsZero() || j.Category != models.CategoryIT || j.GhostingRate != -1 {
				t.Fatalf("job %+v", j)
			}
			if tc.name == "remoteok" && j.ApplyURL != "https://remoteok.com/jobs/1" {
				t.Fatal("provider backlink not preserved")
			}
		})
	}
}

func TestAdzunaCredentialsMockAndSchema(t *testing.T) {
	s := &Adzuna{Country: "gb"}
	jobs, err := s.Fetch(context.Background(), Query{})
	if err != nil || len(jobs) != 1 || !jobs[0].IsMock || jobs[0].Platform != "adzuna-mock" {
		t.Fatalf("mock %v %v", jobs, err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gb/search/1" || r.URL.Query().Get("app_id") != "id" || r.URL.Query().Get("app_key") != "key" || r.URL.Query().Get("what") != "go & sql" {
			t.Errorf("unexpected request %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"results":[{"id":"1","title":"Go Engineer","redirect_url":"https://adzuna.test/jobs/1","salary_min":40000,"salary_max":60000,"location":{"display_name":"London"},"company":{"display_name":"Acme"},"category":{"label":"IT Jobs"}}]}`)
	}))
	defer server.Close()
	s = &Adzuna{HTTP: NewHTTP(server.Client(), 1, time.Second), URL: server.URL, AppID: "id", AppKey: "key", Country: "gb"}
	mockKey := (&Adzuna{URL: server.URL, Country: "gb"}).CacheKey(Query{})
	if mockKey == s.CacheKey(Query{}) {
		t.Fatal("mock cache collides with live feed")
	}
	jobs, err = s.Fetch(context.Background(), Query{Keyword: "go & sql"})
	if err != nil || len(jobs) != 1 || jobs[0].Currency != "GBP" || jobs[0].IsMock {
		t.Fatalf("live %v %v", jobs, err)
	}
}

func TestGitHubBothLabelsDedupAndPullRequestExclusion(t *testing.T) {
	labels := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		labels[q] = true
		if r.Header.Get("Authorization") != "Bearer test-token" || !strings.Contains(q, "is:issue") {
			t.Error("incorrect GitHub request")
		}
		fmt.Fprint(w, `{"items":[{"id":1,"title":"Go bug bounty $500","state":"open","html_url":"https://github.com/acme/repo/issues/1","repository_url":"https://api.github.com/repos/acme/repo","created_at":"2026-10-01T00:00:00Z"},{"id":2,"title":"PR","state":"open","html_url":"https://github.com/acme/repo/pull/2","pull_request":{}}]}`)
	}))
	defer server.Close()
	s := &GitHub{HTTP: NewHTTP(server.Client(), 2, time.Second), URL: server.URL, Token: "test-token"}
	jobs, err := s.Fetch(context.Background(), Query{})
	if err != nil || len(jobs) != 1 || len(labels) != 2 {
		t.Fatalf("%v %v labels=%v", jobs, err, labels)
	}
	if j := jobs[0]; j.CompensationAmount != 500 || j.CompensationPeriod != "task" || j.Company != "acme/repo" {
		t.Fatalf("job %+v", j)
	}
}

func TestHackerNewsDiscoveryAndDeletedComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/whoishiring.json":
			fmt.Fprint(w, `{"submitted":[10,11,12]}`)
		case "/item/10.json":
			fmt.Fprint(w, `{"id":10,"type":"story","title":"Ask HN: Who is hiring? (September 2026)","time":100,"kids":[90]}`)
		case "/item/11.json":
			fmt.Fprint(w, `{"id":11,"type":"story","title":"Ask HN: Who wants to be hired?","time":300}`)
		case "/item/12.json":
			fmt.Fprint(w, `{"id":12,"type":"story","title":"Ask HN: Who is hiring? (October 2026)","time":200,"kids":[20,21,22,23]}`)
		case "/item/20.json":
			fmt.Fprint(w, `{"id":20,"type":"comment","parent":12,"by":"alice","time":201,"text":"Acme | Go Engineer | Remote | USD 100k/year<p>Go SQL &amp; Docker"}`)
		case "/item/21.json":
			fmt.Fprint(w, `{"id":21,"type":"comment","parent":12,"deleted":true}`)
		case "/item/22.json":
			fmt.Fprint(w, `{"id":22,"type":"comment","parent":20,"text":"Nested reply"}`)
		case "/item/23.json":
			fmt.Fprint(w, `{"id":23,"type":"comment","parent":12,"dead":true,"text":"Dead"}`)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	s := &HackerNews{HTTP: NewHTTP(server.Client(), 2, time.Second), URL: server.URL, Workers: 2, MaxComments: 10}
	jobs, err := s.Fetch(context.Background(), Query{})
	if err != nil || len(jobs) != 1 {
		t.Fatalf("jobs=%v err=%v", jobs, err)
	}
	if j := jobs[0]; j.Company != "Acme" || j.CompensationAmount != 100000 || j.Category != models.CategoryIT {
		t.Fatalf("job %+v", j)
	}
}

func TestWeb3RiskAndUnconfirmedRewards(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tasks":[{"id":"1","title":"Testnet","kind":"testnet","url":"https://example.com/1","reward":{"amount":1000,"currency":"USD","guaranteed":false},"audit":{"status":"passed"}},{"id":"2","title":"Bounty","kind":"bounty","url":"https://example.com/2","reward":{"amount":500,"currency":"USD","guaranteed":true},"audit":{"status":"passed"}},{"id":"3","title":"Deposit","url":"https://example.com/3","requires_deposit":true,"audit":{"status":"passed"}},{"id":"4","title":"Failed audit","url":"https://example.com/4","audit":{"status":"failed"}}]}`)
	}))
	defer server.Close()
	s := &Web3{HTTP: NewHTTP(server.Client(), 1, time.Second), URL: server.URL}
	jobs, err := s.Fetch(context.Background(), Query{})
	if err != nil || len(jobs) != 4 {
		t.Fatalf("%v %v", jobs, err)
	}
	if jobs[0].CompensationAmount != 0 || jobs[0].RiskStatus != models.RiskYellow || jobs[1].CompensationAmount != 500 || jobs[1].RiskStatus != models.RiskGreen || jobs[2].RiskStatus != models.RiskRed || jobs[3].RiskStatus != models.RiskRed {
		t.Fatalf("risk classification %+v", jobs)
	}
	mock, err := (&Web3{}).Fetch(context.Background(), Query{})
	if err != nil || len(mock) == 0 || !mock[0].IsMock {
		t.Fatal("mock provenance missing")
	}
}

func TestForumRSS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<rss version="2.0"><channel><title>Gigs</title><item><title>Go developer $60/hour</title><link>https://example.com/job</link><description>&lt;p&gt;Go SQL&lt;/p&gt;</description><pubDate>Thu, 01 Oct 2026 12:00:00 +0000</pubDate></item></channel></rss>`)
	}))
	defer server.Close()
	s := &Forum{HTTP: NewHTTP(server.Client(), 1, time.Second), URL: server.URL}
	jobs, err := s.Fetch(context.Background(), Query{})
	if err != nil || len(jobs) != 1 || jobs[0].CompensationAmount != 60 || jobs[0].CreatedAt.IsZero() {
		t.Fatalf("%v %v", jobs, err)
	}
}

func TestHTTPStatusCancellationMalformedAndLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rate":
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(429)
		case "/bad":
			fmt.Fprint(w, `{"jobs":`)
		case "/null":
			fmt.Fprint(w, `null`)
		case "/large":
			fmt.Fprint(w, strings.Repeat("x", MaxBodyBytes+1))
		case "/wait":
			<-r.Context().Done()
		default:
			fmt.Fprint(w, `{}`)
		}
	}))
	defer server.Close()
	h := NewHTTP(server.Client(), 2, time.Second)
	var target any
	err := h.GetJSON(context.Background(), server.URL+"/rate", nil, &target)
	var status *StatusError
	if !errors.As(err, &status) || status.StatusCode != 429 || status.RetryAfter != 2*time.Minute {
		t.Fatalf("rate status %v", err)
	}
	for _, path := range []string{"/bad", "/null", "/large"} {
		if err := h.GetJSON(context.Background(), server.URL+path, nil, &target); err == nil {
			t.Fatal("invalid payload accepted", path)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := h.GetJSON(ctx, server.URL+"/wait", nil, &target); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline %v", err)
	}
}

func TestHTTPBoundsAndCredentialRedirectProtection(t *testing.T) {
	var active, peak atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := active.Add(1)
		defer active.Add(-1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()
	h := NewHTTP(server.Client(), 2, time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var v any
			if err := h.GetJSON(context.Background(), server.URL, nil, &v); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if peak.Load() > 2 {
		t.Fatal("HTTP concurrency limit exceeded")
	}
	var targetCalls atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls.Add(1); fmt.Fprint(w, `{}`) }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"?app_key=secret", http.StatusFound)
	}))
	defer redirect.Close()
	var v any
	if err := h.GetJSON(context.Background(), redirect.URL+"?app_key=secret", nil, &v); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatal("secret redirect not blocked or leaked")
	}
	if targetCalls.Load() != 0 {
		t.Fatal("credentials sent to another host")
	}
}
