package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	ConfigDir              string   `json:"-"`
	CacheDir               string   `json:"cache_dir"`
	Workers                int      `json:"workers"`
	HTTPTimeoutSeconds     int      `json:"http_timeout_seconds"`
	CircuitFailures        int      `json:"circuit_failures"`
	CircuitCooldownSeconds int      `json:"circuit_cooldown_seconds"`
	AdzunaAppID            string   `json:"adzuna_app_id,omitempty"`
	AdzunaAppKey           string   `json:"adzuna_app_key,omitempty"`
	AdzunaCountry          string   `json:"adzuna_country"`
	GitHubToken            string   `json:"-"` // environment only; never persisted
	HNThreadID             int64    `json:"hn_thread_id,omitempty"`
	HNMaxComments          int      `json:"hn_max_comments"`
	Web3FeedURL            string   `json:"web3_feed_url,omitempty"`
	ForumFeedURLs          []string `json:"forum_feed_urls,omitempty"`
	Skills                 []string `json:"skills"`
	ProfileText            string   `json:"profile_text,omitempty"`
	TargetHourlyUSD        float64  `json:"target_hourly_usd"`
}

func DefaultConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	configRoot := os.Getenv("XDG_CONFIG_HOME")
	if !filepath.IsAbs(configRoot) {
		configRoot = filepath.Join(home, ".config")
	}
	cacheRoot := os.Getenv("XDG_CACHE_HOME")
	if !filepath.IsAbs(cacheRoot) {
		cacheRoot = filepath.Join(home, ".cache")
	}
	return Config{
		ConfigDir: filepath.Join(configRoot, "hexajobs"), CacheDir: filepath.Join(cacheRoot, "hexajobs"),
		Workers: 4, HTTPTimeoutSeconds: 15, CircuitFailures: 3, CircuitCooldownSeconds: 60,
		AdzunaCountry: "gb", HNMaxComments: 100, Skills: []string{}, TargetHourlyUSD: 30,
	}, nil
}

// LoadConfig overlays a JSON file onto defaults, then applies secret environment overrides.
// A missing file is normal on first use. Invalid configuration is never silently reset.
func LoadConfig(path string) (Config, error) {
	cfg, err := DefaultConfig()
	if err != nil {
		return cfg, err
	}
	if path == "" {
		path = filepath.Join(cfg.ConfigDir, "config.json")
	}
	f, err := os.Open(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return cfg, err
	}
	if err == nil {
		defer f.Close()
		data, readErr := io.ReadAll(io.LimitReader(f, (1<<20)+1))
		if readErr != nil {
			return cfg, readErr
		}
		if len(data) > 1<<20 || !bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
			return cfg, errors.New("config must be a JSON object under 1 MiB")
		}
		d := json.NewDecoder(bytes.NewReader(data))
		d.DisallowUnknownFields()
		if err := d.Decode(&cfg); err != nil {
			return cfg, fmt.Errorf("decode config: %w", err)
		}
		var extra any
		if err := d.Decode(&extra); err != io.EOF {
			return cfg, errors.New("config must contain one JSON object")
		}
	}
	cfg.ConfigDir = filepath.Dir(path)
	if value, ok := os.LookupEnv("ADZUNA_APP_ID"); ok {
		cfg.AdzunaAppID = value
	}
	if value, ok := os.LookupEnv("ADZUNA_APP_KEY"); ok {
		cfg.AdzunaAppKey = value
	}
	cfg.GitHubToken = os.Getenv("GITHUB_TOKEN")
	cfg.AdzunaCountry = strings.ToLower(strings.TrimSpace(cfg.AdzunaCountry))
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if c.Workers < 1 || c.Workers > 32 {
		return errors.New("workers must be between 1 and 32")
	}
	if c.HTTPTimeoutSeconds < 1 || c.HTTPTimeoutSeconds > 120 {
		return errors.New("http_timeout_seconds must be between 1 and 120")
	}
	if c.CircuitFailures < 1 || c.CircuitFailures > 100 || c.CircuitCooldownSeconds < 1 || c.CircuitCooldownSeconds > 86400 {
		return errors.New("circuit breaker settings must be positive")
	}
	if c.HNMaxComments < 1 || c.HNMaxComments > 500 || c.HNThreadID < 0 {
		return errors.New("invalid Hacker News limits")
	}
	if c.TargetHourlyUSD <= 0 || c.TargetHourlyUSD > 1e6 || math.IsNaN(c.TargetHourlyUSD) {
		return errors.New("target_hourly_usd must be positive and finite")
	}
	if c.CacheDir == "" {
		return errors.New("cache_dir is required")
	}
	if c.AdzunaAppID != "" && c.AdzunaAppKey == "" {
		return errors.New("adzuna_app_key is required when app_id is set")
	}
	if len(c.AdzunaCountry) != 2 || strings.Trim(c.AdzunaCountry, "abcdefghijklmnopqrstuvwxyz") != "" {
		return errors.New("adzuna_country must be a lowercase two-letter country code")
	}
	return nil
}

func (c Config) HTTPTimeout() time.Duration { return time.Duration(c.HTTPTimeoutSeconds) * time.Second }

// SaveConfig is explicit: merely constructing the engine never writes credentials.
func SaveConfig(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, data)
}
