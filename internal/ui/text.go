package ui

import (
	"math"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rudeops/rudeclaude/internal/activity"
	"github.com/rudeops/rudeclaude/internal/gfx"
	"github.com/rudeops/rudeclaude/internal/theme"
)

const textWidth = 80

func style(hex string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)) }

var (
	hiStyle     = style(theme.WhiteHex)
	boldStyle   = style(theme.WhiteHex).Bold(true)
	greyStyle   = style(theme.GreyHex)
	dimStyle    = style(theme.DimHex)
	labelStyle  = style(theme.DimHex).Bold(true)
	yellowStyle = style(theme.YellowHex)
	ochreStyle  = style(theme.OchreHex)
	greenStyle  = style(theme.GreenHex)
	redStyle    = style(theme.RedHex)
	trackStyle  = style(theme.TrackHex)
)

type textModel struct {
	*core
	width, height int
}

func NewText(interval time.Duration, demo bool) tea.Model {
	return textModel{core: newCore(interval, demo)}
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(pollInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m textModel) fetchCmd(maxAge time.Duration) tea.Cmd {
	return func() tea.Msg { return fetchReport(maxAge) }
}

func (m textModel) Init() tea.Cmd {
	m.refresh()
	if m.demo {
		return tick()
	}
	m.loading = true
	return tea.Batch(m.fetchCmd(m.interval), tick())
}

func (m textModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "r":
			if !m.loading && !m.demo {
				m.loading = true
				return m, m.fetchCmd(0)
			}
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		m.now = time.Time(msg)
		m.refresh()
		if m.fetchDue() {
			m.loading = true
			return m, tea.Batch(m.fetchCmd(m.interval), tick())
		}
		return m, tick()
	case reportMsg:
		m.setReport(msg)
	}
	return m, nil
}

func (m textModel) View() string {
	status := dimStyle.Render(m.status())
	if a := m.alert(); a != "" {
		status = redStyle.Render(a)
	}
	lines := []string{spread(boldStyle.Render("rudeclaude")+"  "+greyStyle.Render("Claude Code"), status), ""}

	for _, l := range m.limits() {
		lines = append(lines,
			spread(labelStyle.Render(l.label)+"  "+dimStyle.Render(l.reset), boldStyle.Render(strconv.Itoa(int(math.Round(l.used))))+greyStyle.Render(" %")),
			bar(l.used, l.elapsed), "")
	}

	replies, output, cache := m.today()
	lines = append(lines,
		labelStyle.Render("AUJOURD'HUI")+"   "+
			boldStyle.Render(replies)+greyStyle.Render(" réponses · ")+
			boldStyle.Render(output)+greyStyle.Render(" tokens générés · ")+
			boldStyle.Render(cache+" %")+greyStyle.Render(" servis par le cache"),
		"",
		spread(labelStyle.Render("ACTIVITÉ · 60 MIN"), greyStyle.Render(m.activityText())),
		sparkline(m.snap.Minutes[:]), "")

	lines = append(lines, labelStyle.Render("SESSIONS"))
	sessions := m.sessions()
	if len(sessions) == 0 {
		lines = append(lines, dimStyle.Render("aucune session active"))
	}
	for _, s := range sessions {
		left := sessionDot(s) + " " + boldStyle.Render(s.project) + "  " + dimStyle.Render(s.detail)
		doing := greyStyle
		switch {
		case s.asking:
			doing = yellowStyle
		case s.state == activity.Idle:
			doing = dimStyle
		}
		right := doing.Render(s.doing) + "  " + greyStyle.Render(s.context) + "  " + dimStyle.Render(s.age)
		lines = append(lines, spread(left, right))
	}

	var tools []string
	for _, t := range m.tools() {
		tools = append(tools, greyStyle.Render(t.Name)+" "+hiStyle.Render(strconv.Itoa(t.Count)))
	}
	footer := ""
	if len(tools) > 0 {
		footer = dimStyle.Render("outils · 1 h  ") + strings.Join(tools, "  ")
	}
	lines = append(lines, "", footer, spread(dimStyle.Render("r rafraîchir · q quitter"), dimStyle.Render(m.credits())))

	link := "\x1b]8;;" + gfx.SignatureURL + "\x1b\\" + dimStyle.Render("rudeops.com") + "\x1b]8;;\x1b\\"
	sig := dimStyle.Render("propulsé par ") + yellowStyle.Bold(true).Render("RudeOps") + dimStyle.Render("   ·   ") + link
	lines = append(lines, "", lipgloss.PlaceHorizontal(textWidth, lipgloss.Center, sig))

	for i, l := range lines {
		lines[i] = l + strings.Repeat(" ", max(0, textWidth-lipgloss.Width(l)))
	}
	s := strings.Join(lines, "\n")
	if m.width == 0 {
		return s
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s)
}

func bar(used, elapsed float64) string {
	filled := int(used/100*textWidth + 0.5)
	mark := -1
	if elapsed >= 0 {
		mark = min(textWidth-1, int(elapsed*textWidth))
	}
	fill := style(fillHex(used))
	var b strings.Builder
	for i := range textWidth {
		ch, st := "─", trackStyle
		if i < filled {
			ch, st = "━", fill
		}
		if i == mark {
			st = hiStyle
		}
		b.WriteString(st.Render(ch))
	}
	return b.String()
}

func sparkline(values []float64) string {
	levels := []rune("▁▂▃▄▅▆▇█")
	peak := 0.0
	for _, v := range values {
		peak = math.Max(peak, v)
	}
	var b strings.Builder
	for i, v := range values {
		if v <= 0 || peak == 0 {
			b.WriteString(trackStyle.Render("▁"))
			continue
		}
		n := len(levels)
		lvl := min(n-1, int(v/peak*float64(n-1)+0.5))
		st := yellowStyle
		if i < len(values)*2/3 {
			st = ochreStyle
		}
		b.WriteString(st.Render(string(levels[lvl])))
	}
	return b.String()
}

func sessionDot(s sessionInfo) string {
	switch {
	case s.asking:
		return yellowStyle.Render("●")
	case s.state == activity.Working:
		return greenStyle.Render("●")
	case s.state == activity.Waiting:
		return greyStyle.Render("●")
	}
	return dimStyle.Render("●")
}

func fillHex(used float64) string {
	switch {
	case used >= 90:
		return theme.RedHex
	case used >= 70:
		return theme.OrangeHex
	default:
		return theme.YellowHex
	}
}

func spread(left, right string) string {
	gap := max(1, textWidth-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
}
