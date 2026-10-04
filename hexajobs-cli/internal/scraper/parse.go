package scraper

import (
	"encoding/json"
	"html"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

var tagsRE = regexp.MustCompile(`<[^>]*>`)
var scriptsRE = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(?:script|style)>`)

// PlainText extracts searchable text, never executable HTML or terminal controls.
func PlainText(s string) string {
	s = html.UnescapeString(tagsRE.ReplaceAllString(scriptsRE.ReplaceAllString(s, " "), " "))
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

var knownSkills = []string{"go", "golang", "python", "javascript", "typescript", "react", "node.js", "sql", "postgresql", "docker", "kubernetes", "aws", "rust", "solidity", "ethereum", "blockchain", "c++", "c#", "java", "sales", "marketing", "copywriting", "design", "customer support", "data analysis"}

func containsTerm(text, term string) bool {
	start := 0
	for start < len(text) {
		i := strings.Index(text[start:], term)
		if i < 0 {
			return false
		}
		i += start
		end := i + len(term)
		boundary := func(b byte) bool { return !((b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '+' || b == '#') }
		if (i == 0 || boundary(text[i-1])) && (end == len(text) || boundary(text[end])) {
			return true
		}
		start = end
	}
	return false
}

func ExtractSkills(text string) []string {
	text = strings.ToLower(PlainText(text))
	found := map[string]bool{}
	for _, skill := range knownSkills {
		if containsTerm(text, skill) {
			if skill == "golang" {
				skill = "go"
			}
			found[skill] = true
		}
	}
	out := make([]string, 0, len(found))
	for skill := range found {
		out = append(out, skill)
	}
	sort.Strings(out)
	return out
}

func classify(text string) string {
	text = strings.ToLower(PlainText(text))
	for _, term := range []string{"web3", "solidity", "blockchain", "ethereum", "defi", "testnet"} {
		if containsTerm(text, term) {
			return models.CategoryWeb3
		}
	}
	for _, term := range []string{"software", "developer", "engineer", "devops", "data science", "it jobs", "security", "programming", "qa", "sysadmin"} {
		if containsTerm(text, term) {
			return models.CategoryIT
		}
	}
	return models.CategoryNonIT
}

// numeric accepts APIs which inconsistently encode salaries or IDs as strings.
type numeric float64

func (n *numeric) UnmarshalJSON(b []byte) error {
	if string(b) == "null" || string(b) == `""` {
		*n = 0
		return nil
	}
	var value json.Number
	if err := json.Unmarshal(b, &value); err != nil {
		return err
	}
	v, err := strconv.ParseFloat(string(value), 64)
	*n = numeric(v)
	return err
}

type identifier string

func (id *identifier) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	*id = identifier(strings.Trim(string(b), `"`))
	return nil
}

func parseDate(value string) time.Time {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02", time.RFC1123Z, time.RFC1123, time.RFC822Z} {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func setSalary(job *models.JobListing, low, high float64, currency, period string) {
	if math.IsNaN(low) || math.IsInf(low, 0) || math.IsNaN(high) || math.IsInf(high, 0) || low < 0 || high < 0 {
		return
	}
	if low == 0 {
		low = high
	}
	if high == 0 {
		high = low
	}
	if high < low {
		return
	}
	if low == 0 {
		return
	}
	job.CompensationAmount = low/2 + high/2
	job.Currency = strings.ToUpper(currency)
	job.CompensationPeriod = period
	job.CompensationRaw = strconv.FormatFloat(low, 'f', -1, 64) + " - " + strconv.FormatFloat(high, 'f', -1, 64) + " " + job.Currency + "/" + period
}

var moneyRE = regexp.MustCompile(`(?i)(USD|EUR|JPY|IDR|GBP|AUD|CAD|SGD|INR|BRL|NZD|ZAR|CHF|PLN|CNY|US\$|A\$|C\$|\$|€|£|¥|Rp)\s*([0-9][0-9.,]*\s*[kKmM]?)\b(?:\s*[-–]\s*(?:USD|EUR|JPY|IDR|GBP|\$|€|£|¥|Rp)?\s*([0-9][0-9.,]*\s*[kKmM]?)\b)?`)
var suffixMoneyRE = regexp.MustCompile(`(?i)\b([0-9][0-9.,]*\s*[kKmM]?)(?:\s*[-–]\s*([0-9][0-9.,]*\s*[kKmM]?))?\s*(USD|EUR|JPY|IDR|GBP)\b`)

// ParseCompensation is deliberately conservative: explicit currency required;
// token rewards and ambiguous free text stay unknown instead of becoming USD.
func ParseCompensation(job *models.JobListing, raw, defaultPeriod string) {
	job.CompensationRaw = PlainText(raw)
	text := strings.ToLower(job.CompensationRaw)
	if strings.Contains(text, "usdt") || strings.Contains(text, "usdc") {
		return
	}
	match := moneyRE.FindStringSubmatch(job.CompensationRaw)
	if len(match) == 0 {
		if m := suffixMoneyRE.FindStringSubmatch(job.CompensationRaw); len(m) > 0 {
			match = []string{m[0], m[3], m[1], m[2]}
		}
	}
	if len(match) == 0 {
		return
	}
	currency := strings.ToUpper(match[1])
	switch currency {
	case "$", "US$":
		currency = "USD"
	case "€":
		currency = "EUR"
	case "£":
		currency = "GBP"
	case "¥":
		currency = "JPY"
	case "RP":
		currency = "IDR"
	case "A$":
		currency = "AUD"
	case "C$":
		currency = "CAD"
	}
	parse := func(s string) float64 {
		s = strings.ToLower(strings.TrimSpace(s))
		multiplier := 1.0
		if strings.HasSuffix(s, "k") {
			multiplier = 1000
			s = strings.TrimSpace(strings.TrimSuffix(s, "k"))
		} else if strings.HasSuffix(s, "m") {
			multiplier = 1e6
			s = strings.TrimSpace(strings.TrimSuffix(s, "m"))
		}
		// Commas with 3 trailing digits and Indonesian/Japanese grouped dots are thousands separators.
		if i := strings.LastIndex(s, ","); i >= 0 {
			if len(s)-i-1 == 3 {
				s = strings.ReplaceAll(s, ",", "")
			} else {
				s = strings.ReplaceAll(s, ".", "")
				s = strings.ReplaceAll(s, ",", ".")
			}
		}
		if (currency == "IDR" || currency == "JPY") && strings.Contains(s, ".") {
			pieces := strings.Split(s, ".")
			grouped := true
			for _, piece := range pieces[1:] {
				if len(piece) != 3 {
					grouped = false
				}
			}
			if grouped {
				s = strings.Join(pieces, "")
			}
		}
		v, _ := strconv.ParseFloat(s, 64)
		return v * multiplier
	}
	low, high := parse(match[2]), parse(match[3])
	if high == 0 {
		high = low
	}
	// Common abbreviated range: $80-100k.
	if high >= 1000 && low > 0 && low < 1000 && strings.ContainsAny(match[3], "kKmM") && !strings.ContainsAny(match[2], "kKmM") {
		if strings.ContainsAny(match[3], "mM") {
			low *= 1e6
		} else {
			low *= 1000
		}
	}
	period := defaultPeriod
	for _, entry := range []struct {
		period string
		terms  []string
	}{
		{"hour", []string{"/hour", "/hr", "per hour", "hourly"}}, {"day", []string{"/day", "per day", "daily"}}, {"week", []string{"/week", "per week", "weekly"}}, {"month", []string{"/month", "per month", "monthly", "/mo"}}, {"year", []string{"/year", "per year", "annual", "yearly", "/yr"}},
	} {
		for _, term := range entry.terms {
			if strings.Contains(text, term) {
				period = entry.period
				break
			}
		}
	}
	setSalary(job, low, high, currency, period)
	job.CompensationRaw = PlainText(raw)
}
