package activity

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type State int

const (
	Waiting State = iota
	Working
	Idle
)

const (
	idleAfter   = 15 * time.Minute
	visibleFor  = time.Hour
	historySpan = time.Hour
)

type Session struct {
	ID, Project, Branch, Model string
	State                      State
	Tool                       string
	Thinking                   bool
	Interrupted                bool
	Since                      time.Time
	Context                    int
}

type Totals struct {
	Replies                               int
	Input, Output, CacheRead, CacheCreate int
}

func (t Totals) CacheRate() float64 {
	in := t.Input + t.CacheCreate + t.CacheRead
	if in == 0 {
		return 0
	}
	return float64(t.CacheRead) / float64(in)
}

type ToolCount struct {
	Name  string
	Count int
}

type Snapshot struct {
	Sessions []Session
	Minutes  [60]float64
	Rate     float64
	Tools    []ToolCount
	Today    Totals
}

type point struct {
	at     time.Time
	tokens int
}

type toolCall struct {
	at   time.Time
	name string
}

type Tracker struct {
	root     string
	offsets  map[string]int64
	seen     map[string]bool
	sessions map[string]*Session
	points   []point
	tools    []toolCall
	day      time.Time
	since    time.Time
	today    Totals
	started  bool
}

func NewTracker() *Tracker {
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".claude")
	}
	return &Tracker{
		root:     filepath.Join(dir, "projects"),
		offsets:  map[string]int64{},
		seen:     map[string]bool{},
		sessions: map[string]*Session{},
	}
}

func (t *Tracker) Poll(now time.Time) {
	midnight := startOfDay(now)
	if !midnight.Equal(t.day) {
		t.day, t.today = midnight, Totals{}
		t.seen = map[string]bool{}
	}
	t.since = midnight
	if h := now.Add(-historySpan); h.Before(t.since) {
		t.since = h
	}

	_ = filepath.WalkDir(t.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		size, offset := info.Size(), t.offsets[path]
		switch {
		case !t.started && info.ModTime().Before(t.since):
			t.offsets[path] = size
		case size < offset:
			t.offsets[path] = 0
		case size > offset:
			t.read(path, offset, size)
		}
		return nil
	})
	t.started = true
	t.prune(now)
}

func (t *Tracker) read(path string, offset, size int64) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	buf := make([]byte, size-offset)
	n, _ := f.ReadAt(buf, offset)
	buf = buf[:n]
	end := bytes.LastIndexByte(buf, '\n')
	if end < 0 {
		return
	}
	t.offsets[path] = offset + int64(end) + 1
	for line := range bytes.SplitSeq(buf[:end], []byte{'\n'}) {
		t.handle(line)
	}
}

type entry struct {
	Type        string    `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
	SessionID   string    `json:"sessionId"`
	Cwd         string    `json:"cwd"`
	GitBranch   string    `json:"gitBranch"`
	IsSidechain bool      `json:"isSidechain"`
	IsMeta      bool      `json:"isMeta"`
	Message     *struct {
		ID         string          `json:"id"`
		Model      string          `json:"model"`
		StopReason *string         `json:"stop_reason"`
		Usage      *usage          `json:"usage"`
		Content    json.RawMessage `json:"content"`
	} `json:"message"`
}

type usage struct {
	Input       int `json:"input_tokens"`
	Output      int `json:"output_tokens"`
	CacheCreate int `json:"cache_creation_input_tokens"`
	CacheRead   int `json:"cache_read_input_tokens"`
}

type block struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
	Text string `json:"text"`
}

func (t *Tracker) handle(line []byte) {
	if !bytes.Contains(line, []byte(`"type":"assistant"`)) && !bytes.Contains(line, []byte(`"type":"user"`)) {
		return
	}
	var e entry
	if json.Unmarshal(line, &e) != nil || e.Message == nil || e.SessionID == "" || e.IsMeta {
		return
	}
	if e.Timestamp.Before(t.since) {
		return
	}
	var blocks []block
	_ = json.Unmarshal(e.Message.Content, &blocks)

	switch e.Type {
	case "assistant":
		t.assistant(e, blocks)
	case "user":
		t.user(e, blocks)
	}
}

func (t *Tracker) assistant(e entry, blocks []block) {
	m := e.Message
	if m.Model == "<synthetic>" {
		return
	}
	if m.Usage != nil && m.ID != "" && !t.seen[m.ID] {
		t.seen[m.ID] = true
		t.points = append(t.points, point{e.Timestamp, m.Usage.Output})
		if !e.Timestamp.Before(t.day) {
			t.today.Replies++
			t.today.Input += m.Usage.Input
			t.today.Output += m.Usage.Output
			t.today.CacheRead += m.Usage.CacheRead
			t.today.CacheCreate += m.Usage.CacheCreate
		}
	}
	lastTool := ""
	for _, b := range blocks {
		if b.Type == "tool_use" && !t.seen[b.ID] {
			t.seen[b.ID] = true
			t.tools = append(t.tools, toolCall{e.Timestamp, b.Name})
		}
		if b.Type == "tool_use" {
			lastTool = b.Name
		}
	}
	if e.IsSidechain {
		return
	}

	s := t.session(e)
	s.Model = m.Model
	if m.Usage != nil {
		s.Context = m.Usage.Input + m.Usage.CacheRead + m.Usage.CacheCreate + m.Usage.Output
	}
	s.Thinking, s.Interrupted = false, false
	switch {
	case m.StopReason == nil:
		s.State, s.Tool = Working, ""
	case *m.StopReason == "tool_use":
		if lastTool != "" {
			s.Tool = lastTool
		}
		s.State = Working
		if s.Tool == "AskUserQuestion" {
			s.State = Waiting
		}
	default:
		s.State, s.Tool = Waiting, ""
	}
}

func (t *Tracker) user(e entry, blocks []block) {
	if e.IsSidechain {
		return
	}
	s := t.session(e)
	s.Tool = ""
	s.State, s.Thinking, s.Interrupted = Working, true, false
	if len(blocks) == 0 {
		var text string
		_ = json.Unmarshal(e.Message.Content, &text)
		blocks = []block{{Type: "text", Text: text}}
	}
	for _, b := range blocks {
		if b.Type == "text" && strings.HasPrefix(b.Text, "[Request interrupted") {
			s.State, s.Thinking, s.Interrupted = Waiting, false, true
		}
	}
}

func (t *Tracker) session(e entry) *Session {
	s, ok := t.sessions[e.SessionID]
	if !ok {
		s = &Session{ID: e.SessionID}
		t.sessions[e.SessionID] = s
	}
	if e.Cwd != "" {
		s.Project = filepath.Base(e.Cwd)
	}
	if e.GitBranch != "" && e.GitBranch != "HEAD" {
		s.Branch = e.GitBranch
	}
	if e.Timestamp.After(s.Since) {
		s.Since = e.Timestamp
	}
	return s
}

func (t *Tracker) prune(now time.Time) {
	cut := now.Add(-historySpan)
	points := t.points[:0]
	for _, p := range t.points {
		if !p.at.Before(cut) {
			points = append(points, p)
		}
	}
	t.points = points
	tools := t.tools[:0]
	for _, c := range t.tools {
		if !c.at.Before(cut) {
			tools = append(tools, c)
		}
	}
	t.tools = tools
	for id, s := range t.sessions {
		if now.Sub(s.Since) > visibleFor {
			delete(t.sessions, id)
		}
	}
}

func (t *Tracker) Snapshot(now time.Time) Snapshot {
	snap := Snapshot{Today: t.today}

	for _, p := range t.points {
		age := int(now.Sub(p.at) / time.Minute)
		if age >= 0 && age < len(snap.Minutes) {
			snap.Minutes[len(snap.Minutes)-1-age] += float64(p.tokens)
			if age < 5 {
				snap.Rate += float64(p.tokens) / 5
			}
		}
	}

	counts := map[string]int{}
	for _, c := range t.tools {
		counts[c.name]++
	}
	for name, n := range counts {
		snap.Tools = append(snap.Tools, ToolCount{name, n})
	}
	sort.Slice(snap.Tools, func(i, j int) bool {
		a, b := snap.Tools[i], snap.Tools[j]
		return a.Count > b.Count || a.Count == b.Count && a.Name < b.Name
	})

	for _, s := range t.sessions {
		c := *s
		if now.Sub(c.Since) > idleAfter {
			c.State = Idle
		}
		snap.Sessions = append(snap.Sessions, c)
	}
	sort.Slice(snap.Sessions, func(i, j int) bool { return snap.Sessions[i].Since.After(snap.Sessions[j].Since) })
	return snap
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
