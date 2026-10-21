package ui

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rudeops/rudeclaude/internal/activity"
	"github.com/rudeops/rudeclaude/internal/gfx"
	"github.com/rudeops/rudeclaude/internal/i18n"
	"github.com/rudeops/rudeclaude/internal/theme"
)

const (
	defaultTextWidth = 80
	maxTextWidth     = 120
	minTextWidth     = 50
	bigLimitsWidth   = 72
	sideLimitsWidth  = 90
)

func style(hex string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)) }

var (
	hiStyle     = style(theme.WhiteHex)
	boldStyle   = style(theme.WhiteHex).Bold(true)
	greyStyle   = style(theme.GreyHex)
	dimStyle    = style(theme.DimHex)
	labelStyle  = style(theme.DimHex).Bold(true)
	yellowStyle = style(theme.YellowHex)
	ochreStyle  = style(theme.OchreHex)
	amberStyle  = style(theme.AmberHex)
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

type block struct {
	name  string
	lines []string
}

var (
	dropOrder = []string{"signature", "tools", "big", "rtk", "products", "keys", "today", "activity"}
	gapOrder  = []string{"limits", "sessions", "footer", "activity", "today", "rtk", "products", "signature"}
)

func (m textModel) View() string {
	w := textWidth(m.width)
	if w < minTextWidth {
		return m.tooSmall()
	}
	blocks := m.fit(w)
	if blocks == nil {
		return m.tooSmall()
	}
	var lines []string
	for i, b := range blocks {
		if b.gap && i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, b.lines...)
	}
	for i, l := range lines {
		lines[i] = l + strings.Repeat(" ", max(0, w-lipgloss.Width(l)))
	}
	return m.place(strings.Join(lines, "\n"))
}

func textWidth(cols int) int {
	switch {
	case cols == 0:
		return defaultTextWidth
	case cols < 64:
		return cols - 2
	default:
		return min(maxTextWidth, cols-4)
	}
}

func (m textModel) place(s string) string {
	if m.width == 0 {
		return s
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, s)
}

func (m textModel) tooSmall() string {
	lines := strings.Split(lipgloss.NewStyle().Width(max(1, m.width)).Align(lipgloss.Center).Render(i18n.T.TooSmall), "\n")
	if m.height > 0 {
		lines = lines[:min(len(lines), m.height)]
	}
	return m.place(dimStyle.Render(strings.Join(lines, "\n")))
}

type spaced struct {
	block
	gap bool
}

type shape struct {
	chart, bars int
	dropped     []string
}

func (m textModel) fit(w int) []spaced {
	if m.height > 0 {
		for _, size := range [][2]int{{4, 2}, {3, 2}, {2, 1}} {
			if s, ok := m.try(w, shape{chart: size[0], bars: size[1]}, -1); ok {
				return s
			}
		}
	}
	for _, gaps := range []int{3, 0} {
		for n := range len(dropOrder) + 1 {
			if s, ok := m.try(w, shape{chart: 1, bars: 1, dropped: dropOrder[:n]}, gaps); ok {
				return s
			}
		}
	}
	return nil
}

func (m textModel) try(w int, sh shape, gaps int) ([]spaced, bool) {
	blocks := m.blocks(w, sh)
	content := 0
	for _, b := range blocks {
		content += len(b.lines)
	}
	if gaps < 0 {
		gaps = len(blocks) - 1
	}
	if m.height > 0 && content+min(gaps, len(blocks)-1) > m.height {
		return nil, false
	}
	return withGaps(blocks, m.height-content), true
}

func withGaps(blocks []block, spare int) []spaced {
	out := make([]spaced, len(blocks))
	for i, b := range blocks {
		out[i].block = b
	}
	if spare < 0 {
		spare = len(blocks)
	}
	for _, name := range gapOrder {
		for i := range out {
			if spare > 0 && i > 0 && out[i].name == name && !out[i].gap {
				out[i].gap = true
				spare--
			}
		}
	}
	return out
}

func (m textModel) blocks(w int, sh shape) []block {
	drop := func(name string) bool { return slices.Contains(sh.dropped, name) }
	var out []block
	add := func(name string, lines []string) {
		if !drop(name) && len(lines) > 0 {
			out = append(out, block{name, lines})
		}
	}
	add("header", []string{m.header(w)})
	add("limits", m.limitLines(w, !drop("big")))
	add("today", m.todayLines(w))
	add("products", m.productLines(w))
	add("activity", m.activityLines(w, sh.chart))
	add("rtk", m.rtkLines(w, sh.bars))
	add("sessions", m.sessionLines(w))
	add("footer", m.footerLines(w, drop("tools"), drop("keys")))
	add("signature", []string{m.signatureLine(w)})
	return out
}

func (m textModel) header(w int) string {
	status := dimStyle.Render(m.status())
	if a := m.alert(); a != "" {
		status = redStyle.Render(a)
	}
	title := boldStyle.Render("rudeclaude") + " " + greyStyle.Render(Version)
	if Version == "" || lipgloss.Width(title)+lipgloss.Width(status)+2 > w {
		title = boldStyle.Render("rudeclaude")
	}
	if avail := w - lipgloss.Width(title) - 2; lipgloss.Width(status) > avail {
		status = dimStyle.Render(truncate(m.status(), avail))
		if a := m.alert(); a != "" {
			status = redStyle.Render(truncate(a, avail))
		}
	}
	return justify(title, status, w)
}

func (m textModel) limitLines(w int, big bool) []string {
	limits := m.limits()
	if w < bigLimitsWidth {
		var out []string
		for i, l := range limits {
			if i > 0 {
				out = append(out, "")
			}
			out = append(out, limitHead(l, w), bar(l.used, l.elapsed, w))
		}
		return out
	}
	left := (w - 4) / 2
	widths := [2]int{left, w - 4 - left}
	var halves [2][]string
	for i, l := range limits {
		if big || w < sideLimitsWidth {
			halves[i] = bigLimit(l, widths[i])
		} else {
			halves[i] = []string{limitHead(l, widths[i]), bar(l.used, l.elapsed, widths[i])}
		}
	}
	out := make([]string, len(halves[0]))
	for j := range out {
		out[j] = pad(halves[0][j], widths[0]) + "    " + halves[1][j]
	}
	return out
}

func limitHead(l limitInfo, w int) string {
	pct := boldStyle.Render(strconv.Itoa(int(math.Round(l.used)))) + greyStyle.Render(i18n.T.Percent)
	return justify(labelStyle.Render(l.label)+"  "+dimStyle.Render(l.reset), pct, w)
}

var bigDigits = [10][3]string{
	{"█▀█", "█ █", "▀▀▀"},
	{"▀█ ", " █ ", "▀▀▀"},
	{"▀▀█", "█▀▀", "▀▀▀"},
	{"▀▀█", " ▀█", "▀▀▀"},
	{"█ █", "▀▀█", "  ▀"},
	{"█▀▀", "▀▀█", "▀▀▀"},
	{"█▀▀", "█▀█", "▀▀▀"},
	{"▀▀█", "  █", "  ▀"},
	{"█▀█", "█▀█", "▀▀▀"},
	{"█▀█", "▀▀█", "▀▀▀"},
}

func bigNumber(n int) [3]string {
	var rows [3]string
	for i, d := range strconv.Itoa(max(0, n)) {
		for r := range rows {
			if i > 0 {
				rows[r] += " "
			}
			rows[r] += bigDigits[d-'0'][r]
		}
	}
	return rows
}

func bigLimit(l limitInfo, w int) []string {
	digits := bigNumber(int(math.Round(l.used)))
	dw := lipgloss.Width(digits[0])
	return []string{
		justify(labelStyle.Render(l.label), hiStyle.Render(digits[0])+"  ", w),
		justify(dimStyle.Render(l.reset), hiStyle.Render(digits[1])+greyStyle.Render(" %"), w),
		bar(l.used, l.elapsed, w-dw-4) + "  " + hiStyle.Render(digits[2]) + "  ",
	}
}

func (m textModel) todayLines(w int) []string {
	t := i18n.T
	replies, output, cache := m.today()
	values := []string{
		boldStyle.Render(replies) + greyStyle.Render(" "+m.repliesLabel()),
		boldStyle.Render(output) + greyStyle.Render(" "+t.Generated),
		boldStyle.Render(percent(cache)) + greyStyle.Render(" "+t.FromCache),
	}
	sep := greyStyle.Render(" · ")
	if one := labelStyle.Render(t.Today) + "   " + strings.Join(values, sep); lipgloss.Width(one) <= w {
		return []string{one}
	}
	return append([]string{labelStyle.Render(t.Today)}, flow(values, sep, w)...)
}

func (m textModel) productLines(w int) []string {
	products := m.products()
	if len(products) == 0 {
		return nil
	}
	var legend []string
	for _, p := range products {
		legend = append(legend, productStyle(p.key).Render("●")+" "+greyStyle.Render(p.name)+" "+hiStyle.Render(percent(strconv.Itoa(int(math.Round(p.percent))))))
	}
	label := labelStyle.Render(i18n.T.ByProduct)
	var out []string
	if one := strings.Join(legend, "  "); lipgloss.Width(label)+lipgloss.Width(one)+2 <= w {
		out = []string{justify(label, one, w)}
	} else {
		out = append([]string{label}, flow(legend, "  ", w)...)
	}
	return append(out, productBar(products, w))
}

func (m textModel) activityLines(w, rows int) []string {
	head := justify(labelStyle.Render(i18n.T.Activity), greyStyle.Render(m.activityText()), w)
	return append([]string{head}, sparkline(m.snap.Minutes[:], w, rows)...)
}

func (m textModel) rtkLines(w, rows int) []string {
	r, ok := m.rtkInfo()
	if !ok {
		return nil
	}
	t := i18n.T
	bars := make([][]string, rows)
	var letters []string
	for i, v := range r.days {
		st, lt := ochreStyle, dimStyle
		if i == len(r.days)-1 {
			st, lt = amberStyle, hiStyle
		}
		for row, cell := range column(v, slices.Max(r.days[:]), rows, st) {
			bars[row] = append(bars[row], cell)
		}
		letters = append(letters, lt.Render(r.letters[i]))
	}
	chart := strings.Join(bars[0], " ")
	saved := boldStyle.Render(r.today) + greyStyle.Render(" "+t.SavedToday)
	if all := saved + greyStyle.Render(" · ") + boldStyle.Render(r.total) + greyStyle.Render(" "+t.AllTime); lipgloss.Width(all)+lipgloss.Width(chart)+2 <= w {
		saved = all
	}
	out := []string{
		justify(labelStyle.Render(t.RTK), greyStyle.Render(percent(r.rate)+" "+t.OnAverage), w),
		justify(saved, chart, w),
	}
	for _, row := range bars[1:] {
		out = append(out, justify("", strings.Join(row, " "), w))
	}
	return append(out, justify("", strings.Join(letters, " "), w))
}

func (m textModel) sessionLines(w int) []string {
	out := []string{labelStyle.Render(i18n.T.Sessions)}
	sessions := m.sessions()
	if len(sessions) == 0 {
		out = append(out, dimStyle.Render(i18n.T.NoSession))
	}
	for _, s := range sessions {
		out = append(out, sessionLine(s, w))
	}
	for len(out) < maxSessions+1 {
		out = append(out, "")
	}
	return out
}

func sessionLine(s sessionInfo, w int) string {
	doingStyle := greyStyle
	switch {
	case s.asking:
		doingStyle = yellowStyle
	case s.state == activity.Idle:
		doingStyle = dimStyle
	}
	tail := "  " + s.context + "  " + s.age
	doing := truncate(s.doing, w-len([]rune(tail))-len([]rune(s.project))-6)
	right := doingStyle.Render(doing) + "  " + greyStyle.Render(s.context) + "  " + dimStyle.Render(s.age)
	avail := w - lipgloss.Width(right) - 2
	project := truncate(s.project, avail-2)
	left := sessionDot(s) + " " + boldStyle.Render(project)
	if room := avail - lipgloss.Width(left) - 2; room >= 6 {
		left += "  " + dimStyle.Render(truncate(s.detail, room))
	}
	return justify(left, right, w)
}

func (m textModel) footerLines(w int, noTools, noKeys bool) []string {
	var out []string
	if tools := m.tools(); len(tools) > 0 && !noTools {
		line := dimStyle.Render(i18n.T.Tools)
		for _, t := range tools {
			item := "  " + greyStyle.Render(t.Name) + " " + hiStyle.Render(strconv.Itoa(t.Count))
			if lipgloss.Width(line+item) > w {
				break
			}
			line += item
		}
		out = append(out, line)
	}
	keys, credits := dimStyle.Render(i18n.T.Keys), dimStyle.Render(m.credits())
	if noKeys || lipgloss.Width(keys)+lipgloss.Width(credits)+2 > w {
		keys = ""
	}
	if keys != "" || m.credits() != "" {
		out = append(out, justify(keys, credits, w))
	}
	return out
}

func (m textModel) signatureLine(w int) string {
	link := "\x1b]8;;" + gfx.SignatureURL + "\x1b\\" + dimStyle.Render("rudeops.com") + "\x1b]8;;\x1b\\"
	sig := dimStyle.Render(i18n.T.PoweredBy) + yellowStyle.Bold(true).Render("RudeOps") + dimStyle.Render("   ·   ") + link
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, sig)
}

func bar(used, elapsed float64, w int) string {
	filled := int(used/100*float64(w) + 0.5)
	mark := -1
	if elapsed >= 0 {
		mark = min(w-1, int(elapsed*float64(w)))
	}
	fill := style(fillHex(used))
	var b strings.Builder
	for i := range w {
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

var levels = []rune("▁▂▃▄▅▆▇█")

func column(v, peak float64, rows int, st lipgloss.Style) []string {
	eighths := 0
	if v > 0 && peak > 0 {
		eighths = int(v/peak*float64(rows*8-1)+0.5) + 1
	}
	out := make([]string, rows)
	for r := range out {
		switch fill := eighths - (rows-1-r)*8; {
		case fill > 0:
			out[r] = st.Render(string(levels[min(8, fill)-1]))
		case r == rows-1:
			out[r] = trackStyle.Render("▁")
		default:
			out[r] = " "
		}
	}
	return out
}

func sparkline(values []float64, w, rows int) []string {
	n := len(values)
	peak := slices.Max(values)
	lines := make([]strings.Builder, rows)
	for i := range w {
		pos := float64(i) * float64(n-1) / float64(max(1, w-1))
		j := int(pos)
		v := values[j]
		if j+1 < n {
			v += (values[j+1] - v) * (pos - float64(j))
		}
		st := yellowStyle
		if pos < float64(n)*2/3 {
			st = ochreStyle
		}
		if i == w-1 {
			st = amberStyle
		}
		for r, cell := range column(v, peak, rows, st) {
			lines[r].WriteString(cell)
		}
	}
	out := make([]string, rows)
	for r := range lines {
		out[r] = lines[r].String()
	}
	return out
}

func productStyle(key string) lipgloss.Style {
	switch key {
	case "claude_code":
		return yellowStyle
	case "chat":
		return style(theme.BlueHex)
	case "cowork":
		return style(theme.PurpleHex)
	}
	return dimStyle
}

func productBar(products []productInfo, w int) string {
	total := 0.0
	for _, p := range products {
		total += p.percent
	}
	avail := w - (len(products) - 1)
	var parts []string
	used := 0
	for i, p := range products {
		n := max(1, int(math.Round(p.percent/total*float64(avail))))
		if i == len(products)-1 {
			n = max(1, avail-used)
		}
		used += n
		parts = append(parts, productStyle(p.key).Render(strings.Repeat("━", n)))
	}
	return strings.Join(parts, " ")
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

func justify(left, right string, w int) string {
	gap := max(1, w-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
}

func pad(s string, w int) string {
	return s + strings.Repeat(" ", max(0, w-lipgloss.Width(s)))
}

func flow(parts []string, sep string, w int) []string {
	var out []string
	line := ""
	for _, p := range parts {
		switch {
		case line == "":
			line = p
		case lipgloss.Width(line+sep+p) <= w:
			line += sep + p
		default:
			out = append(out, line)
			line = p
		}
	}
	return append(out, line)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return ""
	}
	return strings.TrimRight(string(r[:n-1]), " ·") + "…"
}
