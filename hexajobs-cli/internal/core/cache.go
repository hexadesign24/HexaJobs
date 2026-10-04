package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"hexajobs.dev/hexajobs-cli/internal/models"
)

const CacheTTL = 30 * time.Minute
const CacheRetention = 7 * 24 * time.Hour
const maxCacheBytes = 32 << 20

var ErrCacheMiss = errors.New("cache miss or expired")

type cacheEntry struct {
	Version   int                 `json:"version"`
	WrittenAt time.Time           `json:"written_at"`
	Jobs      []models.JobListing `json:"jobs"`
}

type Cache struct {
	dir string
	mu  sync.RWMutex
	now func() time.Time
}

func NewCache(dir string) (*Cache, error) {
	if dir == "" {
		return nil, errors.New("empty cache directory")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &Cache{dir: dir, now: time.Now}, nil
}

func (c *Cache) path(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(sum[:])+".json")
}

func (c *Cache) Get(key string) ([]models.JobListing, error) { return c.GetWithTTL(key, CacheTTL) }

func (c *Cache) GetWithTTL(key string, ttl time.Duration) ([]models.JobListing, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	path := c.path(key)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxCacheBytes {
		return nil, ErrCacheMiss
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var entry cacheEntry
	d := json.NewDecoder(io.LimitReader(f, maxCacheBytes+1))
	if err := d.Decode(&entry); err != nil {
		return nil, ErrCacheMiss
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return nil, ErrCacheMiss
	}
	age := c.now().Sub(entry.WrittenAt)
	if entry.Version != 1 || age < 0 || age >= ttl {
		return nil, ErrCacheMiss
	}
	return entry.Jobs, nil
}

func (c *Cache) Put(key string, jobs []models.JobListing) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := json.Marshal(cacheEntry{Version: 1, WrittenAt: c.now().UTC(), Jobs: jobs})
	if err != nil {
		return err
	}
	if len(data) > maxCacheBytes {
		return errors.New("cache entry exceeds size limit")
	}
	return atomicWrite(c.path(key), data)
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".hexajobs-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
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
	return os.Rename(name, path)
}
