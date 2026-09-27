package usage

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestRetryAfter(t *testing.T) {
	if got := retryAfter("90"); got != 90*time.Second {
		t.Errorf("secondes : %v, attendu 1m30s", got)
	}
	if got := retryAfter(time.Now().Add(5 * time.Minute).UTC().Format(http.TimeFormat)); got < 4*time.Minute || got > 5*time.Minute {
		t.Errorf("date HTTP : %v, attendu ~5 min", got)
	}
	if got := retryAfter(""); got != 2*time.Minute {
		t.Errorf("sans en-tête : %v, attendu 2 min", got)
	}
}

func TestGetUsesSharedCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	body := []byte(`{"five_hour":{"utilization":42}}`)

	saveCache(cacheFile{At: time.Now().Add(-10 * time.Second), Body: body})
	res, err := Get(t.Context(), time.Minute)
	if err != nil || res.Report == nil || res.Report.FiveHour.Utilization != 42 {
		t.Fatalf("cache frais : %+v, %v", res, err)
	}

	saveCache(cacheFile{At: time.Now().Add(-time.Hour), BlockedUntil: time.Now().Add(time.Minute), Body: body})
	res, err = Get(t.Context(), time.Minute)
	var rl *RateLimitError
	if !errors.As(err, &rl) || !errors.Is(err, ErrRateLimited) || rl.RetryAfter <= 0 {
		t.Fatalf("pause : erreur %v, attendu une RateLimitError", err)
	}
	if res.Report == nil || res.Report.FiveHour.Utilization != 42 {
		t.Errorf("pause : %+v, attendu la dernière réponse connue", res)
	}
}
