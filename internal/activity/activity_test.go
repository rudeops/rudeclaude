package activity

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func appendTo(t *testing.T, path string, lines ...string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, l := range lines {
		if _, err := f.WriteString(l); err != nil {
			t.Fatal(err)
		}
	}
}

func answer(ts time.Time, id, stop, content string) string {
	stopJSON := "null"
	if stop != "" {
		stopJSON = `"` + stop + `"`
	}
	return fmt.Sprintf(`{"type":"assistant","timestamp":%q,"sessionId":"s1","cwd":"/home/moi/projets/demo","gitBranch":"main",`+
		`"message":{"id":%q,"model":"claude-opus-5-5","stop_reason":%s,`+
		`"usage":{"input_tokens":2,"output_tokens":800,"cache_creation_input_tokens":1000,"cache_read_input_tokens":100000},`+
		`"content":%s}}`+"\n", ts.Format(time.RFC3339Nano), id, stopJSON, content)
}

func userLine(ts time.Time, content string) string {
	return fmt.Sprintf(`{"type":"user","timestamp":%q,"sessionId":"s1","cwd":"/home/moi/projets/demo","message":{"role":"user","content":%s}}`+"\n",
		ts.Format(time.RFC3339Nano), content)
}

func TestTracker(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	path := filepath.Join(dir, "projects", "demo", "s1.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now()

	appendTo(t, path,
		userLine(now.Add(-2*time.Minute), `"salut"`),
		answer(now.Add(-time.Minute), "msg_1", "tool_use", `[{"type":"text","text":"ok"}]`),
		answer(now.Add(-time.Minute), "msg_1", "tool_use", `[{"type":"tool_use","id":"toolu_1","name":"Bash"}]`),
	)

	old := filepath.Join(dir, "projects", "demo", "s0.jsonl")
	appendTo(t, old, fmt.Sprintf(`{"type":"assistant","timestamp":%q,"sessionId":"s0","cwd":"/x","message":{"id":"msg_0","model":"claude-opus-5-5","stop_reason":"tool_use","usage":{"output_tokens":5},"content":[{"type":"tool_use","id":"toolu_0","name":"Read"}]}}`+"\n",
		now.Add(-2*time.Hour).Format(time.RFC3339Nano)))

	tr := NewTracker()
	tr.Poll(now)
	snap := tr.Snapshot(now)

	oldToday := 0
	if !now.Add(-2 * time.Hour).Before(startOfDay(now)) {
		oldToday = 1
	}
	if snap.Today.Replies != 1+oldToday {
		t.Errorf("réponses = %d, attendu %d (doublon ignoré)", snap.Today.Replies, 1+oldToday)
	}
	if len(snap.Sessions) != 1 {
		t.Fatalf("%d sessions, attendu 1", len(snap.Sessions))
	}
	s := snap.Sessions[0]
	if s.Project != "demo" || s.Branch != "main" || s.State != Working || s.Tool != "Bash" {
		t.Errorf("session = %+v, attendu demo/main en train d'exécuter Bash", s)
	}
	if want := 2 + 100000 + 1000 + 800; s.Context != want {
		t.Errorf("contexte = %d, attendu %d", s.Context, want)
	}
	if snap.Minutes[len(snap.Minutes)-2] != 800 {
		t.Errorf("activité il y a une minute = %v, attendu 800 tokens générés", snap.Minutes[len(snap.Minutes)-2])
	}
	if len(snap.Tools) != 1 || snap.Tools[0] != (ToolCount{"Bash", 1}) {
		t.Errorf("outils = %+v, attendu Bash×1", snap.Tools)
	}

	appendTo(t, path, userLine(now, `[{"type":"tool_result","tool_use_id":"toolu_1"}]`))
	tr.Poll(now)
	if s := tr.Snapshot(now).Sessions[0]; s.State != Working || !s.Thinking {
		t.Errorf("après le résultat : %+v, attendu en réflexion", s)
	}
	appendTo(t, path, answer(now, "msg_2", "end_turn", `[{"type":"text","text":"fini"}]`), `{"type":"assis`)
	tr.Poll(now)
	snap = tr.Snapshot(now)
	if s := snap.Sessions[0]; s.State != Waiting {
		t.Errorf("après la fin : %+v, attendu en attente", s)
	}
	if snap.Today.Replies != 2+oldToday {
		t.Errorf("réponses = %d, attendu %d (ligne incomplète ignorée)", snap.Today.Replies, 2+oldToday)
	}

	appendTo(t, path, `tant"}`+"\n", userLine(now, `[{"type":"text","text":"[Request interrupted by user]"}]`))
	tr.Poll(now)
	if s := tr.Snapshot(now).Sessions[0]; s.State != Waiting || !s.Interrupted {
		t.Errorf("après l'interruption : %+v, attendu en attente et interrompue", s)
	}

	if s := tr.Snapshot(now.Add(20 * time.Minute)).Sessions[0]; s.State != Idle {
		t.Errorf("après 20 min : %+v, attendu inactive", s)
	}
}
