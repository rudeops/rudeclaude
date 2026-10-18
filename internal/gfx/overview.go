package gfx

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/fogleman/gg"

	"github.com/rudeops/rudeclaude/internal/i18n"
)

const (
	SignatureURL = "https://www.rudeops.com"

	maxScale = 2.5
	margin   = 28.0

	wideMinW, wideMaxW, wideH = 920.0, 1280.0, 724.0
	tallMinW, tallMaxW, tallH = 560.0, 760.0, 1166.0

	bodySize    = 14.0
	labelSize   = 12.0
	projectSize = 16.0
	doingSize   = 14.5
	gaugeW      = 80.0
)

func Fit(pxW, pxH float64) Options {
	wide := fit(pxW, pxH, wideMinW, wideMaxW, wideH)
	tall := fit(pxW, pxH, tallMinW, tallMaxW, tallH)
	if tall.Scale > wide.Scale {
		tall.Tall = true
		return tall
	}
	return wide
}

func fit(pxW, pxH, minW, maxW, h float64) Options {
	s := min(maxScale, pxW/minW, pxH/h)
	return Options{Scale: s, Width: min(maxW, pxW/s)}
}

func (o Options) Size() (w, h float64) {
	if o.Tall {
		return max(tallMinW, min(tallMaxW, o.Width)), tallH
	}
	return max(wideMinW, min(wideMaxW, o.Width)), wideH
}

func SignatureRect(opt Options) (x, y, w, h float64) {
	cw, ch := opt.Size()
	w = signatureWidth(&canvas{gg.NewContext(1, 1), 1})
	return (cw - w) / 2, ch - 16 - 13, w, 17
}

type signaturePart struct {
	s    string
	w    weight
	gold bool
}

func signature() []signaturePart {
	return []signaturePart{
		{i18n.T.PoweredBy, light, false},
		{"RudeOps", medium, true},
		{"   ·   rudeops.com", regular, false},
	}
}

func signatureWidth(dc *canvas) float64 {
	total := 0.0
	for _, p := range signature() {
		total += measure(dc, p.s, p.w, 12.5)
	}
	return total
}

func drawSignature(dc *canvas, w, h float64) {
	x := (w - signatureWidth(dc)) / 2
	for _, p := range signature() {
		c := textLow
		if p.gold {
			c = yellow
		}
		x = text(dc, p.s, x, h-16, p.w, 12.5, c, 0, 0)
	}
}

type Limit struct {
	Label   string
	Used    float64
	Elapsed float64
	Reset   string
}

type Stat struct{ Value, Unit, Label string }

type SessionState int

const (
	Done SessionState = iota
	Asking
	Working
	Idle
)

type Session struct {
	Project, Detail string
	State           SessionState
	Doing           string
	Context         string
	ContextFrac     float64
	Age             string
}

type Tool struct {
	Name  string
	Count int
}

type Product struct {
	Key, Name string
	Percent   float64
}

type RTK struct {
	Today, Total, Rate string
	Days               [7]float64
	Letters            [7]string
}

type Overview struct {
	Version        string
	Updated, Alert string
	Limits         [2]Limit
	Today          []Stat
	Products       []Product
	RTK            *RTK
	Activity       []float64
	ActivityNow    string
	Sessions       []Session
	Tools          []Tool
	Credits        string
	Pulse          float64
}

func RenderOverview(o Overview, opt Options) *image.RGBA {
	w, h := opt.Size()
	dc := &canvas{gg.NewContext(int(math.Round(w*opt.Scale)), int(math.Round(h*opt.Scale))), opt.Scale}
	dc.Scale(opt.Scale, opt.Scale)
	if !opt.Transparent {
		dc.SetColor(bg)
		dc.Clear()
	}
	left, right := margin, w-margin
	drawHeader(dc, o, left, right)
	if opt.Tall {
		drawTall(dc, o, left, right)
	} else {
		drawWide(dc, o, left, right)
	}
	drawSignature(dc, w, h)
	return dc.Image().(*image.RGBA)
}

func drawHeader(dc *canvas, o Overview, left, right float64) {
	const y = 46.0
	x := text(dc, "rudeclaude", left, y, medium, 19, textHi, 0, 0)
	x = text(dc, o.Version, x+11, y, light, 19, textMid, 0, 0)
	status, c := o.Updated, textLow
	if o.Alert != "" {
		status, c = o.Alert, red
	}
	text(dc, ellipsis(dc, status, regular, bodySize, right-x-30), right, y, regular, bodySize, c, 1, 0)
}

func drawWide(dc *canvas, o Overview, left, right float64) {
	colX := left + math.Round((right-left)*0.52)
	ringsW := colX - 30 - left
	for i, l := range o.Limits {
		drawLimit(dc, left+ringsW*(0.25+0.5*float64(i)), 172, l)
	}
	drawToday(dc, o.Today, colX, right, 92)
	drawProducts(dc, o.Products, colX, right, 216)
	separator(dc, left, right, 322)

	const rowY = 354.0
	actRight := right
	if o.RTK != nil {
		actRight = colX - 60
		drawRTK(dc, o.RTK, colX, right, rowY)
		dc.SetColor(border)
		dc.SetLineWidth(1)
		dc.DrawLine(colX-30, rowY-16, colX-30, rowY+82)
		dc.Stroke()
	}
	drawActivity(dc, o, left, actRight, rowY, 64)
	separator(dc, left, right, 458)
	drawSessions(dc, o, left, right, 490, colX, false)
	drawFooter(dc, o, left, right, 684, false)
}

func drawTall(dc *canvas, o Overview, left, right float64) {
	for i, l := range o.Limits {
		drawLimit(dc, left+(right-left)*(0.25+0.5*float64(i)), 172, l)
	}
	separator(dc, left, right, 322)
	drawToday(dc, o.Today, left, right, 354)
	drawProducts(dc, o.Products, left, right, 474)
	separator(dc, left, right, 556)
	chartH := 174.0
	if o.RTK != nil {
		chartH = 60
		drawRTK(dc, o.RTK, left, right, 700)
	}
	drawActivity(dc, o, left, right, 588, chartH)
	separator(dc, left, right, 806)
	drawSessions(dc, o, left, right, 838, 0, true)
	drawFooter(dc, o, left, right, 1104, true)
}

func separator(dc *canvas, left, right, y float64) {
	dc.SetColor(border)
	dc.SetLineWidth(1)
	dc.DrawLine(left, y, right, y)
	dc.Stroke()
}

func label(dc *canvas, s string, x, y float64) {
	text(dc, s, x, y, medium, labelSize, textLow, 0, 2.2)
}

func ellipsis(dc *canvas, s string, w weight, size, maxW float64) string {
	if measure(dc, s, w, size) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && measure(dc, string(r)+"…", w, size) > maxW {
		r = r[:len(r)-1]
	}
	if len(r) == 0 {
		return ""
	}
	return strings.TrimRight(string(r), " ·") + "…"
}

func drawLimit(dc *canvas, cx, cy float64, l Limit) {
	const r, w = 70.0, 7.0
	fill := fillColor(l.Used)

	dc.SetLineCapRound()
	dc.SetLineWidth(w)
	dc.SetColor(track)
	dc.DrawCircle(cx, cy, r)
	dc.Stroke()
	if l.Used > 0 {
		start := -math.Pi / 2
		dc.SetColor(fill)
		dc.DrawArc(cx, cy, r, start, start+2*math.Pi*math.Min(l.Used, 100)/100)
		dc.Stroke()
	}
	if l.Elapsed >= 0 {
		a := -math.Pi/2 + 2*math.Pi*l.Elapsed
		dc.SetColor(textMid)
		dc.DrawCircle(cx+(r+13)*math.Cos(a), cy+(r+13)*math.Sin(a), 2.5)
		dc.Fill()
	}

	num := itoa(l.Used)
	nw := measure(dc, num, light, 46)
	pw := measure(dc, "%", light, 18)
	x := cx - (nw+2+pw)/2
	text(dc, num, x, cy+16, light, 46, textHi, 0, 0)
	text(dc, "%", x+nw+2, cy+16, light, 18, textMid, 0, 0)

	text(dc, l.Label, cx, cy+r+34, medium, labelSize, textMid, 0.5, 2.2)
	text(dc, l.Reset, cx, cy+r+55, regular, bodySize, textLow, 0.5, 0)
}

func drawToday(dc *canvas, stats []Stat, x, right, y float64) {
	label(dc, i18n.T.Today, x, y)
	step := (right - x) / 3
	for i, s := range stats {
		sx := x + float64(i)*step
		vx := text(dc, s.Value, sx, y+52, light, 38, textHi, 0, 0)
		text(dc, s.Unit, vx+3, y+52, light, 17, textMid, 0, 0)
		text(dc, ellipsis(dc, s.Label, regular, bodySize, step-12), sx, y+76, regular, bodySize, textLow, 0, 0)
	}
}

func productColor(key string) color.Color {
	switch key {
	case "claude_code":
		return yellow
	case "chat":
		return blue
	case "cowork":
		return purple
	}
	return textLow
}

func drawProducts(dc *canvas, ps []Product, x, right, y float64) {
	if len(ps) == 0 {
		return
	}
	label(dc, i18n.T.ByProduct, x, y)
	const gap, h = 5.0, 6.0
	barY := y + 20
	total := 0.0
	for _, p := range ps {
		total += p.Percent
	}
	avail := right - x - gap*float64(len(ps)-1)
	bx := x
	for _, p := range ps {
		w := math.Max(h, avail*p.Percent/total)
		dc.SetColor(productColor(p.Key))
		dc.DrawRoundedRectangle(bx, barY-h/2, w, h, h/2)
		dc.Fill()
		bx += w + gap
	}
	lx := x
	for _, p := range ps {
		dc.SetColor(productColor(p.Key))
		dc.DrawCircle(lx+4, barY+25, 4)
		dc.Fill()
		lx = text(dc, p.Name, lx+14, barY+30, regular, bodySize, textMid, 0, 0) + 6
		lx = text(dc, itoa(p.Percent)+i18n.T.Percent, lx, barY+30, medium, bodySize, textHi, 0, 0) + 20
	}
}

func drawActivity(dc *canvas, o Overview, left, right, y, h float64) {
	label(dc, i18n.T.Activity, left, y)
	text(dc, o.ActivityNow, right, y, regular, bodySize, textMid, 1, 0)
	activityBars(dc, left, y+16, right-left, h, o.Activity)
}

func activityBars(dc *canvas, x, y, w, h float64, values []float64) {
	n := len(values)
	if n == 0 {
		return
	}
	peak := 0.0
	for _, v := range values {
		peak = math.Max(peak, v)
	}
	step := w / float64(n)
	bw := math.Max(1, step-1.5)
	for i, v := range values {
		bx := x + float64(i)*step
		bh := 1.5
		if peak > 0 && v > 0 {
			bh = math.Max(3, h*v/peak)
		}
		age := float64(n-1-i) / float64(n)
		c := yellow
		if v == 0 {
			c = track
		}
		dc.SetColor(alpha(c, 1-0.65*age))
		dc.DrawRoundedRectangle(bx, y+h-bh, bw, bh, math.Min(1.5, bw/2))
		dc.Fill()
	}
}

func drawRTK(dc *canvas, r *RTK, x, right, y float64) {
	t := i18n.T
	label(dc, t.RTK, x, y)
	text(dc, r.Rate+t.Percent+" "+t.OnAverage, right, y, regular, bodySize, textMid, 1, 0)

	const step, bw, h = 19.0, 11.0, 48.0
	bx := right - step*7 + (step-bw)/2
	stats := []Stat{{Value: r.Today, Label: t.SavedToday}, {Value: r.Total, Label: t.AllTime}}
	for i, s := range stats {
		sx := x + float64(i)*max(175, min(260, (bx-x)*0.55))
		text(dc, s.Value, sx, y+50, light, 38, textHi, 0, 0)
		text(dc, s.Label, sx, y+74, regular, bodySize, textLow, 0, 0)
	}

	peak := 0.0
	for _, v := range r.Days {
		peak = math.Max(peak, v)
	}
	for i, v := range r.Days {
		cx := bx + float64(i)*step
		bh := 1.5
		var c color.Color = track
		if peak > 0 && v > 0 {
			bh = math.Max(3, h*v/peak)
			c = alpha(yellow, 0.5)
			if i == len(r.Days)-1 {
				c = yellow
			}
		}
		dc.SetColor(c)
		dc.DrawRoundedRectangle(cx, y+60-bh, bw, bh, 1.5)
		dc.Fill()
		lc := textLow
		if i == len(r.Days)-1 {
			lc = textHi
		}
		text(dc, r.Letters[i], cx+bw/2, y+78, medium, 11.5, lc, 0.5, 0)
	}
}

type sessionCols struct {
	left, right, doing, gauge, tokens, age float64
}

func drawSessions(dc *canvas, o Overview, left, right, y, colX float64, tall bool) {
	t := i18n.T
	label(dc, t.Sessions, left, y)
	if len(o.Sessions) == 0 {
		text(dc, t.NoSession, left, y+36, regular, 15, textLow, 0, 0)
		return
	}
	c := sessionCols{left: left, right: right}
	c.age = measure(dc, fmt.Sprintf(t.Ago, fmt.Sprintf(t.Minutes, 59)), regular, bodySize)
	tokW := measure(dc, "000k", regular, bodySize)
	if tall {
		c.gauge = right - tokW - 14 - gaugeW
	} else {
		c.tokens = right - c.age - 18 - tokW
		c.gauge = c.tokens - 14 - gaugeW
		c.doing = min(colX, c.gauge-24-doingWidth(dc))
	}
	for i, s := range o.Sessions {
		if tall {
			drawSessionTall(dc, c, y+36+56*float64(i), s, o.Pulse)
		} else {
			drawSessionWide(dc, c, y+36+38*float64(i), s, o.Pulse)
		}
	}
}

func doingWidth(dc *canvas) float64 {
	t := i18n.T
	w := 0.0
	for _, s := range []string{t.Idle, t.Asking, t.Interrupted, t.Finished, t.Thinking, t.Writing} {
		w = max(w, measure(dc, s, regular, doingSize))
	}
	suffix := " · " + fmt.Sprintf(t.Minutes, 59)
	for _, v := range t.ToolVerbs {
		w = max(w, measure(dc, v+suffix, regular, doingSize))
	}
	return w
}

func stateColors(s SessionState) (dot, doing color.Color) {
	switch s {
	case Working:
		return green, textMid
	case Asking:
		return yellow, yellow
	case Done:
		return textMid, textMid
	}
	return textLow, textLow
}

func drawDot(dc *canvas, x, y float64, s Session, pulse float64) {
	dot, _ := stateColors(s.State)
	if s.State == Working {
		dc.SetColor(alpha(green, 0.25*(1-pulse)))
		dc.DrawCircle(x, y, 5+7*pulse)
		dc.Fill()
	}
	dc.SetColor(dot)
	dc.DrawCircle(x, y, 5)
	dc.Fill()
}

func drawGauge(dc *canvas, x, y, frac float64) {
	dc.SetLineCapRound()
	dc.SetLineWidth(4)
	dc.SetColor(rail)
	dc.DrawLine(x, y, x+gaugeW, y)
	dc.Stroke()
	if frac > 0 {
		dc.SetColor(contextColor(frac))
		dc.DrawLine(x, y, x+gaugeW*math.Min(1, frac), y)
		dc.Stroke()
	}
}

func drawSessionWide(dc *canvas, c sessionCols, y float64, s Session, pulse float64) {
	_, doing := stateColors(s.State)
	drawDot(dc, c.left+5, y-5, s, pulse)
	x := text(dc, ellipsis(dc, s.Project, medium, projectSize, c.doing-c.left-42), c.left+22, y, medium, projectSize, textHi, 0, 0)
	text(dc, ellipsis(dc, s.Detail, regular, bodySize, c.doing-x-31), x+11, y, regular, bodySize, textLow, 0, 0)
	text(dc, ellipsis(dc, s.Doing, regular, doingSize, c.gauge-c.doing-24), c.doing, y, regular, doingSize, doing, 0, 0)
	drawGauge(dc, c.gauge, y-5, s.ContextFrac)
	text(dc, s.Context, c.tokens, y, regular, bodySize, textMid, 0, 0)
	text(dc, s.Age, c.right, y, regular, bodySize, textLow, 1, 0)
}

func drawSessionTall(dc *canvas, c sessionCols, y float64, s Session, pulse float64) {
	_, doing := stateColors(s.State)
	drawDot(dc, c.left+5, y-5, s, pulse)
	x := text(dc, ellipsis(dc, s.Project, medium, projectSize, c.right-c.age-c.left-42), c.left+22, y, medium, projectSize, textHi, 0, 0)
	text(dc, ellipsis(dc, s.Detail, regular, bodySize, c.right-c.age-x-31), x+11, y, regular, bodySize, textLow, 0, 0)
	text(dc, s.Age, c.right, y, regular, bodySize, textLow, 1, 0)

	y += 22
	text(dc, ellipsis(dc, s.Doing, regular, doingSize, c.gauge-c.left-46), c.left+22, y, regular, doingSize, doing, 0, 0)
	drawGauge(dc, c.gauge, y-5, s.ContextFrac)
	text(dc, s.Context, c.right, y, regular, bodySize, textMid, 1, 0)
}

func drawFooter(dc *canvas, o Overview, left, right, y float64, tall bool) {
	credW := measure(dc, o.Credits, regular, bodySize)
	limit := right - credW - 30
	if tall {
		limit = right
	}
	end := left
	if len(o.Tools) > 0 {
		fx := text(dc, i18n.T.Tools, left, y, regular, bodySize, textLow, 0, 0) + 15
		for _, t := range o.Tools {
			count := itoa(float64(t.Count))
			if fx+measure(dc, t.Name, regular, bodySize)+5+measure(dc, count, medium, bodySize) > limit {
				break
			}
			fx = text(dc, t.Name, fx, y, regular, bodySize, textMid, 0, 0) + 5
			end = text(dc, count, fx, y, medium, bodySize, textHi, 0, 0)
			fx = end + 15
		}
	}
	if tall && end+30+credW > right {
		y += 24
	}
	text(dc, o.Credits, right, y, regular, bodySize, textLow, 1, 0)
}
