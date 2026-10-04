package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

const FrankfurterURL = "https://api.frankfurter.app/latest?from=USD"

// JSONGetter is shared by the bounded HTTP ingestion layer and the FX client.
type JSONGetter interface {
	GetJSON(context.Context, string, map[string]string, any) error
}

type CurrencyConverter struct {
	getter    JSONGetter
	endpoint  string
	mu        sync.RWMutex
	refresh   chan struct{}
	rates     map[string]float64 // currency units per USD
	source    string
	date      string
	expires   time.Time
	lastError error
	now       func() time.Time
}

func NewCurrencyConverter(getter JSONGetter, endpoint string, now func() time.Time) *CurrencyConverter {
	if endpoint == "" {
		endpoint = FrankfurterURL
	}
	if now == nil {
		now = time.Now
	}
	return &CurrencyConverter{getter: getter, endpoint: endpoint, now: now, refresh: make(chan struct{}, 1),
		rates: map[string]float64{"USD": 1, "EUR": 0.92, "JPY": 150, "IDR": 16000, "GBP": 0.79}, source: "offline"}
}

// Refresh caches daily reference rates for 30 minutes. On failure, previous rates
// remain available; an initially offline table is explicitly marked as such.
func (c *CurrencyConverter) Refresh(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case c.refresh <- struct{}{}:
		defer func() { <-c.refresh }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.RLock()
	fresh := c.now().Before(c.expires)
	lastError := c.lastError
	c.mu.RUnlock()
	if fresh {
		return lastError
	}
	var payload struct {
		Base  string             `json:"base"`
		Date  string             `json:"date"`
		Rates map[string]float64 `json:"rates"`
	}
	var err error
	if c.getter == nil {
		err = errors.New("FX HTTP client unavailable")
	} else {
		err = c.getter.GetJSON(ctx, c.endpoint, nil, &payload)
	}
	if err == nil {
		if payload.Base != "USD" || len(payload.Rates) == 0 {
			err = errors.New("invalid FX base or empty rates")
		}
		if _, dateErr := time.Parse("2006-01-02", payload.Date); dateErr != nil {
			err = errors.New("invalid FX reference date")
		}
		for code, rate := range payload.Rates {
			if len(code) != 3 || !finitePositive(rate) {
				err = errors.New("invalid FX quote")
				break
			}
		}
		for _, code := range []string{"EUR", "JPY", "IDR"} {
			if !finitePositive(payload.Rates[code]) {
				err = errors.New("missing required FX quote")
			}
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		c.lastError = err
		if c.source == "frankfurter" {
			c.source = "frankfurter-stale"
		}
		c.expires = c.now().Add(time.Minute)
		return err
	}
	payload.Rates["USD"] = 1
	c.rates, c.source, c.date = payload.Rates, "frankfurter", payload.Date
	c.lastError = nil
	c.expires = c.now().Add(30 * time.Minute)
	return nil
}

func finitePositive(v float64) bool { return v > 0 && !math.IsNaN(v) && !math.IsInf(v, 0) }

func (c *CurrencyConverter) ToUSD(amount float64, currency string) (float64, error) {
	if amount < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return 0, errors.New("invalid compensation amount")
	}
	c.mu.RLock()
	rate := c.rates[strings.ToUpper(strings.TrimSpace(currency))]
	c.mu.RUnlock()
	if !finitePositive(rate) {
		return 0, fmt.Errorf("unsupported currency %q", currency)
	}
	value := amount / rate
	if math.IsInf(value, 0) {
		return 0, errors.New("currency conversion overflow")
	}
	return value, nil
}

func (c *CurrencyConverter) Status() (source, date string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.source, c.date
}

func (c *CurrencyConverter) Normalize(job *models.JobListing) {
	job.CompensationUSD = 0
	if !finitePositive(job.CompensationAmount) {
		return
	}
	usd, err := c.ToUSD(job.CompensationAmount, job.Currency)
	if err != nil {
		job.ForexSource = "unsupported"
		return
	}
	job.CompensationUSD = usd
	job.ForexSource, _ = c.Status()
	if job.Currency == "USD" {
		job.ForexSource = "native"
	}
}

// HourlyUSD only compares known employment periods. A task reward has no known
// effort estimate and must never be annualized as if it were a salary.
func HourlyUSD(job models.JobListing) (float64, bool) {
	if !finitePositive(job.CompensationUSD) {
		return 0, false
	}
	divisor := map[string]float64{"hour": 1, "day": 8, "week": 40, "month": 2080.0 / 12, "year": 2080}[job.CompensationPeriod]
	if divisor == 0 {
		return 0, false
	}
	return job.CompensationUSD / divisor, true
}
