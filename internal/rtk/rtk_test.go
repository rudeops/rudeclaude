package rtk

import (
	"math"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	raw := []byte(`{
		"summary": {"total_saved": 155929},
		"daily": [
			{"date": "2026-09-20", "input_tokens": 1000, "saved_tokens": 900},
			{"date": "2026-09-23", "input_tokens": 400, "saved_tokens": 100},
			{"date": "2026-09-28", "input_tokens": 1000, "saved_tokens": 500},
			{"date": "2026-09-29", "input_tokens": 600, "saved_tokens": 300}
		]
	}`)
	now := time.Date(2026, 9, 29, 23, 30, 0, 0, time.FixedZone("CEST", 2*3600))
	s, err := parse(raw, now)
	if err != nil {
		t.Fatal(err)
	}
	if s.Total != 155929 || s.Today() != 300 {
		t.Errorf("total %d, aujourd'hui %d", s.Total, s.Today())
	}
	if s.Days[0].Saved != 100 || s.Days[5].Saved != 500 {
		t.Errorf("journées mal placées : %+v", s.Days)
	}
	if got := s.Rate(); math.Abs(got-0.45) > 1e-9 {
		t.Errorf("taux %v, attendu 0.45", got)
	}
}
