package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Undocumented endpoint. If this breaks one day, it was a feature.
const endpoint = "https://api.anthropic.com/api/oauth/usage"

const (
	FiveHours = 5 * time.Hour
	SevenDays = 7 * 24 * time.Hour
)

var (
	ErrTokenExpired = errors.New("token expiré : lance claude pour le rafraîchir")
	ErrRateLimited  = errors.New("API saturée")
)

type Window struct {
	Utilization float64    `json:"utilization"`
	ResetsAt    *time.Time `json:"resets_at"`
}

type BreakdownRow struct {
	Key         string  `json:"key"`
	DisplayName string  `json:"display_name"`
	Percent     float64 `json:"percent"`
}

type Breakdown struct {
	Rows []BreakdownRow `json:"rows"`
}

type Money struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Exponent    int    `json:"exponent"`
}

func (m Money) Value() float64 {
	v := float64(m.AmountMinor)
	for range m.Exponent {
		v /= 10
	}
	return v
}

type Spend struct {
	Enabled bool   `json:"enabled"`
	Used    *Money `json:"used"`
	Limit   *Money `json:"limit"`
}

type Report struct {
	Spend             *Spend     `json:"spend"`
	FiveHour          *Window    `json:"five_hour"`
	SevenDay          *Window    `json:"seven_day"`
	SevenDayBreakdown *Breakdown `json:"seven_day_breakdown"`
}

type credentials struct {
	ClaudeAiOauth struct {
		AccessToken string `json:"accessToken"`
		ExpiresAt   int64  `json:"expiresAt"`
	} `json:"claudeAiOauth"`
}

var client = &http.Client{Timeout: 10 * time.Second}

var UserAgent = "rudeclaude"

func credentialsPath() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, ".credentials.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", ".credentials.json")
}

func readToken() (string, error) {
	raw, err := os.ReadFile(credentialsPath())
	if err != nil {
		return "", fmt.Errorf("lecture des identifiants Claude Code : %w", err)
	}
	var c credentials
	if err := json.Unmarshal(raw, &c); err != nil {
		return "", fmt.Errorf("identifiants Claude Code illisibles : %w", err)
	}
	if c.ClaudeAiOauth.AccessToken == "" {
		return "", errors.New("aucun token OAuth : connecte-toi avec claude")
	}
	if time.Now().After(time.UnixMilli(c.ClaudeAiOauth.ExpiresAt)) {
		return "", ErrTokenExpired
	}
	return c.ClaudeAiOauth.AccessToken, nil
}

func fetchRaw(ctx context.Context) ([]byte, error) {
	token, err := readToken()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("appel API : %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("lecture de la réponse : %w", err)
		}
		return body, nil
	case http.StatusUnauthorized:
		return nil, ErrTokenExpired
	case http.StatusTooManyRequests:
		return nil, &RateLimitError{RetryAfter: retryAfter(resp.Header.Get("Retry-After"))}
	default:
		return nil, fmt.Errorf("réponse inattendue de l'API : %s", resp.Status)
	}
}

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return "API saturée, nouvel essai dans " + e.RetryAfter.Round(time.Second).String()
}

func (e *RateLimitError) Is(target error) bool { return target == ErrRateLimited }

func retryAfter(h string) time.Duration {
	const fallback = 2 * time.Minute
	if secs, err := strconv.Atoi(strings.TrimSpace(h)); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(h); err == nil && time.Until(t) > 0 {
		return time.Until(t)
	}
	return fallback
}

func decode(body []byte) (*Report, error) {
	var r Report
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("réponse illisible : %w", err)
	}
	return &r, nil
}

type Pace struct {
	Elapsed   float64
	Projected float64
	LimitAt   time.Time
}

func (w *Window) Pace(length time.Duration, now time.Time) (p Pace, ok bool) {
	if w == nil || w.ResetsAt == nil {
		return p, false
	}
	start := w.ResetsAt.Add(-length)
	elapsed := now.Sub(start)
	p.Elapsed = max(0, min(1, float64(elapsed)/float64(length)))
	if elapsed <= 0 {
		return p, true
	}
	p.Projected = w.Utilization / p.Elapsed
	if w.Utilization >= 100 {
		p.LimitAt = now
	} else if p.Projected > 100 {
		p.LimitAt = start.Add(time.Duration(float64(elapsed) * 100 / w.Utilization))
	}
	return p, true
}
