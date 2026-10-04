package core

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PurgeExpired removes only engine-owned files older than seven days.
// It does not recurse or follow symlinks. Count includes successful removals only.
func (c *Cache) PurgeExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return 0
	}
	cutoff, count := c.now().Add(-CacheRetention), 0
	for _, entry := range entries {
		name := entry.Name()
		isCache := len(name) == 69 && strings.HasSuffix(name, ".json")
		if isCache {
			_, err := hex.DecodeString(name[:64])
			isCache = err == nil
		}
		isTemp := strings.HasPrefix(name, ".hexajobs-") && strings.HasSuffix(name, ".tmp")
		if (!isCache && !isTemp) || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || !info.ModTime().Before(cutoff) {
			continue
		}
		if os.Remove(filepath.Join(c.dir, name)) == nil {
			count++
		}
	}
	return count
}

// RunAutoPurge blocks until cancellation. Call it in a caller-owned goroutine;
// no hidden background process or shutdown hook is installed by NewEngine.
func (e *Engine) RunAutoPurge(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.PurgeExpiredCache()
		}
	}
}
