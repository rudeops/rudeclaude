package ui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/rudeops/rudeclaude/internal/activity"
	"github.com/rudeops/rudeclaude/internal/usage"
)

const (
	maxBackoff   = 10 * time.Minute
	pollInterval = 500 * time.Millisecond
	maxSessions  = 4
	maxTools     = 5
)

type core struct {
	interval    time.Duration
	demo        bool
	report      *usage.Report
	err         error
	loading     bool
	fetchedAt   time.Time
	attemptedAt time.Time
	backoff     time.Duration
	now         time.Time
	tracker     *activity.Tracker
	snap        activity.Snapshot
}

func newCore(interval time.Duration, demo bool) *core {
	return &core{interval: interval, demo: demo, now: time.Now(), tracker: activity.NewTracker()}
}

type reportMsg struct {
	res usage.Result
	err error
}

func fetchReport(maxAge time.Duration) reportMsg {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, err := usage.Get(ctx, maxAge)
	return reportMsg{res, err}
}

func (c *core) fetchDue() bool {
	return !c.demo && !c.loading && c.now.Sub(c.attemptedAt) >= c.interval+c.backoff
}

func (c *core) setReport(msg reportMsg) {
	c.loading = false
	c.attemptedAt = time.Now()
	c.err = msg.err
	if msg.res.Report != nil {
		c.report, c.fetchedAt = msg.res.Report, msg.res.At
	}
	c.backoff = 0
	if rl := (*usage.RateLimitError)(nil); errors.As(msg.err, &rl) {
		c.backoff = min(maxBackoff, max(0, rl.RetryAfter-c.interval))
	}
}

func (c *core) rateLimited() bool { return errors.Is(c.err, usage.ErrRateLimited) }

func (c *core) alert() string {
	if c.err == nil || c.rateLimited() && c.report != nil {
		return ""
	}
	if c.rateLimited() {
		return "API saturée, nouvel essai dans " + formatDuration(c.nextFetch().Sub(c.now))
	}
	return c.err.Error()
}

func (c *core) nextFetch() time.Time { return c.attemptedAt.Add(c.interval + c.backoff) }

func (c *core) refresh() {
	if c.demo {
		c.report, c.fetchedAt = demoReport(c.now), c.now
		c.snap = demoSnapshot(c.now)
		return
	}
	c.tracker.Poll(c.now)
	c.snap = c.tracker.Snapshot(c.now)
}

type limitInfo struct {
	label   string
	used    float64
	elapsed float64
	reset   string
}

func (c *core) limits() [2]limitInfo {
	var five, week *usage.Window
	if c.report != nil {
		five, week = c.report.FiveHour, c.report.SevenDay
	}
	return [2]limitInfo{
		c.limit("SESSION · 5 H", five, usage.FiveHours),
		c.limit("SEMAINE", week, usage.SevenDays),
	}
}

func (c *core) limit(label string, w *usage.Window, length time.Duration) limitInfo {
	info := limitInfo{label: label, elapsed: -1}
	switch {
	case c.report == nil && c.err == nil:
		info.reset = "chargement…"
	case w == nil:
		info.reset = "indisponible"
	default:
		info.used = w.Utilization
		if p, ok := w.Pace(length, c.now); ok {
			info.elapsed = p.Elapsed
			info.reset = "reset dans " + formatDuration(w.ResetsAt.Sub(c.now))
		}
	}
	return info
}

type sessionInfo struct {
	project, detail, doing, context, age string
	state                                activity.State
	asking                               bool
	contextFrac                          float64
}

func (c *core) sessions() []sessionInfo {
	var out []sessionInfo
	for _, s := range c.snap.Sessions {
		if len(out) == maxSessions {
			break
		}
		detail := modelName(s.Model)
		if s.Branch != "" {
			detail = s.Branch + " · " + detail
		}
		limit := contextLimit(s.Model)
		if s.Context > limit {
			limit = 1_000_000
		}
		out = append(out, sessionInfo{
			project:     s.Project,
			detail:      detail,
			doing:       doing(s, c.now),
			context:     formatTokens(float64(s.Context)),
			contextFrac: float64(s.Context) / float64(limit),
			age:         formatAge(c.now.Sub(s.Since)),
			state:       s.State,
			asking:      s.State == activity.Waiting && s.Tool == "AskUserQuestion",
		})
	}
	return out
}

func doing(s activity.Session, now time.Time) string {
	switch {
	case s.State == activity.Idle:
		return "inactive"
	case s.Tool == "AskUserQuestion":
		return "te pose une question"
	case s.Interrupted:
		return "interrompu"
	case s.State == activity.Waiting:
		return "terminé"
	case s.Thinking:
		return "réfléchit…"
	case s.Tool == "":
		return "rédige…"
	}
	verb := toolVerbs[s.Tool]
	if verb == "" {
		verb = "utilise " + s.Tool
	}
	if d := now.Sub(s.Since); d > 2*time.Minute {
		verb += " · " + formatDuration(d)
	}
	return verb
}

var toolVerbs = map[string]string{
	"Bash":         "exécute une commande",
	"Read":         "lit un fichier",
	"Edit":         "modifie un fichier",
	"MultiEdit":    "modifie un fichier",
	"Write":        "écrit un fichier",
	"NotebookEdit": "modifie un notebook",
	"Grep":         "cherche dans le code",
	"Glob":         "cherche des fichiers",
	"WebFetch":     "consulte le web",
	"WebSearch":    "cherche sur le web",
	"Agent":        "délègue à un agent",
	"Task":         "délègue à un agent",
	"TodoWrite":    "planifie",
	"Skill":        "charge une compétence",
}

func modelName(id string) string {
	parts := strings.Split(strings.TrimPrefix(id, "claude-"), "-")
	if len(parts) == 0 || parts[0] == "" {
		return id
	}
	name := strings.ToUpper(parts[0][:1]) + parts[0][1:]
	var version []string
	for _, p := range parts[1:] {
		if len(p) > 2 {
			break
		}
		version = append(version, p)
	}
	if len(version) == 0 {
		return name
	}
	return name + " " + strings.Join(version, ".")
}

func contextLimit(model string) int {
	for _, p := range []string{"claude-fable-", "claude-mythos-", "claude-opus-5", "claude-sonnet-5",
		"claude-opus-4-6", "claude-opus-4-7", "claude-opus-4-8", "claude-sonnet-4-6"} {
		if strings.HasPrefix(model, p) {
			return 1_000_000
		}
	}
	return 200_000
}

func (c *core) today() (replies, output, cache string) {
	t := c.snap.Today
	return fmt.Sprint(t.Replies), formatTokens(float64(t.Output)), fmt.Sprint(int(math.Round(t.CacheRate() * 100)))
}

func (c *core) tools() []activity.ToolCount {
	return c.snap.Tools[:min(maxTools, len(c.snap.Tools))]
}

func (c *core) activityText() string {
	if c.snap.Rate < 1 {
		return "au calme"
	}
	return formatTokens(c.snap.Rate) + " tokens/min"
}

func (c *core) credits() string {
	if c.report == nil || c.report.Spend == nil || !c.report.Spend.Enabled || c.report.Spend.Used == nil {
		return ""
	}
	sp := c.report.Spend
	s := "crédits extra · " + formatMoney(*sp.Used)
	if sp.Limit != nil {
		s += " sur " + formatMoney(*sp.Limit)
	}
	return s
}

func (c *core) status() string {
	switch {
	case c.demo:
		return "mode démo"
	case c.fetchedAt.IsZero():
		return "connexion…"
	case c.rateLimited():
		return "mis à jour il y a " + formatDuration(c.now.Sub(c.fetchedAt)) +
			" · API saturée, nouvel essai dans " + formatDuration(c.nextFetch().Sub(c.now))
	default:
		return "mis à jour il y a " + formatDuration(c.now.Sub(c.fetchedAt))
	}
}

func formatDuration(d time.Duration) string {
	d = max(0, d)
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%d j %d h", int(d.Hours())/24, int(d.Hours())%24)
	case d >= time.Hour:
		return fmt.Sprintf("%d h %02d", int(d.Hours()), int(d.Minutes())%60)
	case d >= time.Minute:
		return fmt.Sprintf("%d min", int(d.Minutes()))
	default:
		return fmt.Sprintf("%d s", int(d.Seconds()))
	}
}

func formatAge(d time.Duration) string {
	if d < 30*time.Second {
		return "à l'instant"
	}
	return "il y a " + formatDuration(d)
}

func formatTokens(n float64) string {
	switch {
	case n >= 999_500:
		return strings.Replace(fmt.Sprintf("%.1fM", n/1e6), ".", ",", 1)
	case n >= 999.5:
		return fmt.Sprintf("%.0fk", n/1e3)
	default:
		return fmt.Sprintf("%.0f", n)
	}
}

var currencies = map[string]string{"EUR": "€", "USD": "$", "GBP": "£"}

func formatMoney(m usage.Money) string {
	sym := currencies[m.Currency]
	if sym == "" {
		sym = m.Currency
	}
	v := strings.Replace(fmt.Sprintf("%.*f", m.Exponent, m.Value()), ".", ",", 1)
	return v + " " + sym
}
