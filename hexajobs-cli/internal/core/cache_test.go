package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

func TestCacheTTLAtomicReplacementAndCorruption(t *testing.T) {
	c, err := NewCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	key := "../secret?app_key=private"
	if filepath.Dir(c.path(key)) != c.dir || strings.Contains(c.path(key), "private") {
		t.Fatal("unsafe cache key")
	}
	if err := c.Put(key, []models.JobListing{{ID: "one"}}); err != nil {
		t.Fatal(err)
	}
	if err := c.Put(key, []models.JobListing{{ID: "two"}}); err != nil {
		t.Fatal(err)
	}
	jobs, err := c.Get(key)
	if err != nil || len(jobs) != 1 || jobs[0].ID != "two" {
		t.Fatalf("cache replacement %v %v", jobs, err)
	}
	now = now.Add(CacheTTL - time.Nanosecond)
	if _, err := c.Get(key); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Nanosecond)
	if _, err := c.Get(key); !errors.Is(err, ErrCacheMiss) {
		t.Fatal("TTL boundary not expired")
	}
	if err := os.WriteFile(c.path(key), []byte(`{"jobs":`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(key); !errors.Is(err, ErrCacheMiss) {
		t.Fatal("corrupt cache accepted")
	}
	entries, _ := os.ReadDir(c.dir)
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Fatal("temporary file leaked")
		}
	}
}

func TestPurgeOnlyOwnedOldFiles(t *testing.T) {
	c, err := NewCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	c.now = func() time.Time { return now }
	old := now.Add(-CacheRetention - time.Hour)
	for _, key := range []string{"old", "fresh", "boundary"} {
		if err := c.Put(key, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chtimes(c.path("old"), old, old); err != nil {
		t.Fatal(err)
	}
	boundary := now.Add(-CacheRetention)
	if err := os.Chtimes(c.path("boundary"), boundary, boundary); err != nil {
		t.Fatal(err)
	}
	unrelated := filepath.Join(c.dir, "notes.json")
	if err := os.WriteFile(unrelated, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(unrelated, old, old)
	subdir := filepath.Join(c.dir, "subdir")
	if err := os.Mkdir(subdir, 0700); err != nil {
		t.Fatal(err)
	}
	if got := c.PurgeExpired(); got != 1 {
		t.Fatalf("purged %d, expected 1", got)
	}
	for _, path := range []string{unrelated, c.path("fresh"), c.path("boundary"), subdir} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("removed unrelated/fresh file: %s", path)
		}
	}
}

func TestCacheConcurrentReadWritePurge(t *testing.T) {
	c, err := NewCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 4; n++ {
				if err := c.Put("shared", []models.JobListing{{ID: "ok"}}); err != nil {
					t.Error(err)
				}
				if _, err := c.Get("shared"); err != nil {
					t.Error(err)
				}
				c.PurgeExpired()
			}
		}()
	}
	wg.Wait()
}

func TestConfigDefaultsSecretsAndValidation(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("ADZUNA_APP_ID", "")
	t.Setenv("ADZUNA_APP_KEY", "")
	t.Setenv("GITHUB_TOKEN", "secret-test-token")
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Workers != 4 || cfg.GitHubToken != "secret-test-token" {
		t.Fatal("defaults or secret override missing")
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "secret-test-token") {
		t.Fatal("GitHub token persisted")
	}
	for _, bad := range []string{`null`, `[]`, `{"workers":0}`, `{"unknown":true}`, `{"workers":4} {}`, `{"adzuna_app_id":"id"}`, `{"target_hourly_usd":-1}`} {
		if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		if bad == `{"adzuna_app_id":"id"}` {
			t.Setenv("ADZUNA_APP_ID", "id")
		}
		if _, err := LoadConfig(path); err == nil {
			t.Fatalf("accepted bad config %s", bad)
		}
		t.Setenv("ADZUNA_APP_ID", "")
	}
}

func TestAutoPurgeStopsOnCancellation(t *testing.T) {
	c, err := NewCache(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{cache: c}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { e.RunAutoPurge(ctx, time.Millisecond); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("purge worker did not stop")
	}
}
