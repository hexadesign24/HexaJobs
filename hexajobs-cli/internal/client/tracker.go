package client

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

type HistoryEntry struct {
	Job    models.JobListing `json:"job"`
	Action string            `json:"action"`
	At     time.Time         `json:"at"`
}

type Tracker struct{ Path string }

var historyMu sync.Mutex

const maxHistoryBytes = 32 << 20

func DefaultTracker() (Tracker, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Tracker{}, err
	}
	root := os.Getenv("XDG_CACHE_HOME")
	if !filepath.IsAbs(root) {
		root = filepath.Join(home, ".cache")
	}
	return Tracker{Path: filepath.Join(root, "hexajobs", "history.json")}, nil
}

// LogApplication provides the requested silent API. UI uses Tracker.Log instead
// so it can surface persistence failures in its own status bar, never stdout.
func LogApplication(job models.JobListing, actionType string) {
	tracker, err := DefaultTracker()
	if err == nil {
		_ = tracker.Log(job, actionType)
	}
}

func (t Tracker) Read() ([]HistoryEntry, error) {
	historyMu.Lock()
	defer historyMu.Unlock()
	return t.read()
}
func (t Tracker) read() ([]HistoryEntry, error) {
	if t.Path == "" {
		return nil, errors.New("history path not configured")
	}
	f, err := os.Open(t.Path)
	if errors.Is(err, os.ErrNotExist) {
		return []HistoryEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxHistoryBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxHistoryBytes {
		return nil, errors.New("history exceeds 32 MiB limit")
	}
	var entries []HistoryEntry
	if err = json.Unmarshal(data, &entries); err != nil {
		return nil, errors.New("history JSON is corrupt; original file preserved")
	}
	return entries, nil
}

func (t Tracker) Log(job models.JobListing, action string) error {
	switch action {
	case "opened", "saved", "pitch":
	default:
		return errors.New("invalid history action")
	}
	historyMu.Lock()
	defer historyMu.Unlock()
	entries, err := t.read()
	if err != nil {
		return err
	}
	// Preserve description evidence so ScamShield can rescan saved listings.
	job.SkillsRequired = append([]string(nil), job.SkillsRequired...)
	entries = append(entries, HistoryEntry{Job: job, Action: action, At: time.Now().UTC()})
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxHistoryBytes {
		return errors.New("history is full; existing entries preserved")
	}
	return WritePrivateFile(t.Path, data)
}

// WritePrivateFile atomically replaces an application-owned preferences/history
// file. It does not read, expire, or purge any engine cache entry.
func WritePrivateFile(path string, data []byte) error {
	if path == "" {
		return errors.New("empty storage path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".hexajobs-ui-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}
