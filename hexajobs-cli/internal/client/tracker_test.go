package client

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

func TestTrackerConcurrentAppendAndEvidence(t *testing.T) {
	tracker := Tracker{Path: filepath.Join(t.TempDir(), "history.json")}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := tracker.Log(models.JobListing{ID: "job", Title: "Go", Description: "upfront registration fee"}, "saved"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	entries, err := tracker.Read()
	if err != nil || len(entries) != 12 {
		t.Fatalf("entries=%d err=%v", len(entries), err)
	}
	for _, entry := range entries {
		if entry.Action != "saved" || entry.At.IsZero() || entry.Job.Description != "upfront registration fee" {
			t.Fatal("lost event metadata or scam evidence")
		}
	}
}

func TestTrackerPreservesCorruptHistoryAndSeparatesCache(t *testing.T) {
	root := t.TempDir()
	tracker := Tracker{Path: filepath.Join(root, "history.json")}
	if err := os.WriteFile(tracker.Path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tracker.Log(models.JobListing{}, "saved"); err == nil {
		t.Fatal("corruption ignored")
	}
	data, _ := os.ReadFile(tracker.Path)
	if string(data) != "corrupt" {
		t.Fatal("original history overwritten")
	}
	if err := tracker.Log(models.JobListing{}, "applied"); err == nil {
		t.Fatal("browser open must not become submitted application")
	}
	t.Setenv("XDG_CACHE_HOME", root)
	defaultTracker, err := DefaultTracker()
	if err != nil || defaultTracker.Path != filepath.Join(root, "hexajobs", "history.json") {
		t.Fatalf("default path %s %v", defaultTracker.Path, err)
	}
	LogApplication(models.JobListing{ID: "1"}, "opened")
	entries, err := defaultTracker.Read()
	if err != nil || len(entries) != 1 {
		t.Fatalf("silent logger %v %v", entries, err)
	}
}
