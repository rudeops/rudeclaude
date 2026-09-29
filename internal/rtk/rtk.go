package rtk

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync/atomic"
	"time"
)

const (
	Days         = 7
	pollInterval = 30 * time.Second
	timeout      = 5 * time.Second
)

type Day struct {
	Date         time.Time
	Saved, Input int64
}

type Stats struct {
	Days  [Days]Day
	Total int64
}

func (s *Stats) Today() int64 { return s.Days[Days-1].Saved }

func (s *Stats) Rate() float64 {
	var saved, input int64
	for _, d := range s.Days {
		saved += d.Saved
		input += d.Input
	}
	if input == 0 {
		return 0
	}
	return float64(saved) / float64(input)
}

type report struct {
	Summary struct {
		TotalSaved int64 `json:"total_saved"`
	} `json:"summary"`
	Daily []struct {
		Date  string `json:"date"`
		Input int64  `json:"input_tokens"`
		Saved int64  `json:"saved_tokens"`
	} `json:"daily"`
}

func Load(ctx context.Context, now time.Time) (*Stats, error) {
	path, err := exec.LookPath("rtk")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "gain", "--daily", "--format", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("rtk gain : %w", err)
	}
	return parse(out, now)
}

func parse(raw []byte, now time.Time) (*Stats, error) {
	var r report
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("rtk gain illisible : %w", err)
	}
	s := &Stats{Total: r.Summary.TotalSaved}
	today := now.UTC().Truncate(24 * time.Hour)
	for i := range s.Days {
		s.Days[i].Date = today.AddDate(0, 0, i-(Days-1))
	}
	for _, d := range r.Daily {
		date, err := time.Parse(time.DateOnly, d.Date)
		if err != nil {
			continue
		}
		i := Days - 1 - int(today.Sub(date).Hours()/24)
		if i >= 0 && i < Days {
			s.Days[i].Saved, s.Days[i].Input = d.Saved, d.Input
		}
	}
	return s, nil
}

type Watcher struct {
	stats atomic.Pointer[Stats]
	busy  atomic.Bool
	last  time.Time
}

func (w *Watcher) Poll(now time.Time) {
	if !w.last.IsZero() && now.Sub(w.last) < pollInterval || !w.busy.CompareAndSwap(false, true) {
		return
	}
	w.last = now
	go func() {
		defer w.busy.Store(false)
		s, err := Load(context.Background(), now)
		if err == nil {
			w.stats.Store(s)
		} else if _, missing := err.(*exec.Error); missing {
			w.stats.Store(nil)
		}
	}()
}

func (w *Watcher) Stats() *Stats { return w.stats.Load() }
