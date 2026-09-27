package usage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Result struct {
	Report *Report
	At     time.Time
}

type cacheFile struct {
	At           time.Time       `json:"at"`
	BlockedUntil time.Time       `json:"blocked_until,omitzero"`
	Body         json.RawMessage `json:"body,omitempty"`
}

func cachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "rudeclaude", "usage.json")
}

func loadCache() cacheFile {
	var c cacheFile
	if raw, err := os.ReadFile(cachePath()); err == nil {
		_ = json.Unmarshal(raw, &c)
	}
	return c
}

func saveCache(c cacheFile) {
	raw, err := json.Marshal(c)
	if err != nil {
		return
	}
	path := cachePath()
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil {
		return
	}
	f, err := os.CreateTemp(filepath.Dir(path), "usage-*.tmp")
	if err != nil {
		return
	}
	_, err = f.Write(raw)
	if cerr := f.Close(); err == nil && cerr == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		_ = os.Remove(f.Name())
	}
}

func Get(ctx context.Context, maxAge time.Duration) (Result, error) {
	c := loadCache()
	now := time.Now()
	last := Result{At: c.At}
	if c.Body != nil {
		last.Report, _ = decode(c.Body)
	}
	if last.Report != nil && now.Sub(c.At) < maxAge {
		return last, nil
	}
	if now.Before(c.BlockedUntil) {
		return last, &RateLimitError{RetryAfter: c.BlockedUntil.Sub(now)}
	}

	body, err := fetchRaw(ctx)
	if rl := (*RateLimitError)(nil); errors.As(err, &rl) {
		c.BlockedUntil = now.Add(rl.RetryAfter)
		saveCache(c)
	}
	if err != nil {
		return last, err
	}
	r, err := decode(body)
	if err != nil {
		return last, err
	}
	saveCache(cacheFile{At: now, Body: body})
	return Result{Report: r, At: now}, nil
}
