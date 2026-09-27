package gfx

import (
	"image"
	"math"

	"github.com/fogleman/gg"
)

const (
	OverviewW = 960
	OverviewH = 630
)

const (
	SignatureURL = "https://www.rudeops.com"
	signatureY   = 614.0
)

func SignatureRect() (x, y, w, h float64) {
	dc := &canvas{gg.NewContext(1, 1), 1}
	w = signatureWidth(dc)
	return (OverviewW - w) / 2, signatureY - 13, w, 17
}

var signature = []struct {
	s    string
	w    weight
	gold bool
}{
	{"propulsé par ", light, false},
	{"RudeOps", medium, true},
	{"   ·   rudeops.com", regular, false},
}

func signatureWidth(dc *canvas) float64 {
	total := 0.0
	for _, p := range signature {
		total += measure(dc, p.s, p.w, 12.5)
	}
	return total
}

func drawSignature(dc *canvas) {
	x := (OverviewW - signatureWidth(dc)) / 2
	for _, p := range signature {
		c := textLow
		if p.gold {
			c = yellow
		}
		x = text(dc, p.s, x, signatureY, p.w, 12.5, c, 0, 0)
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

type Overview struct {
	Updated, Alert string
	Limits         [2]Limit
	Today          []Stat
	Activity       []float64
	ActivityNow    string
	Sessions       []Session
	Tools          []Tool
	Credits        string
	Pulse          float64
}

func RenderOverview(o Overview, opt Options) *image.RGBA {
	dc := &canvas{gg.NewContext(int(OverviewW*opt.Scale), int(OverviewH*opt.Scale)), opt.Scale}
	dc.Scale(opt.Scale, opt.Scale)
	if !opt.Transparent {
		dc.SetColor(bg)
		dc.Clear()
	}

	const left, right = 40.0, 920.0

	x := text(dc, "rudeclaude", left, 54, medium, 19, textHi, 0, 0)
	text(dc, "Claude Code", x+11, 54, light, 19, textMid, 0, 0)
	if o.Alert != "" {
		text(dc, o.Alert, right, 54, regular, 13.5, red, 1, 0)
	} else {
		text(dc, o.Updated, right, 54, regular, 13.5, textLow, 1, 0)
	}

	for i, l := range o.Limits {
		drawLimit(dc, 122+float64(i)*208, 180, l)
	}

	const colX = 500.0
	label(dc, "AUJOURD'HUI", colX, 106)
	for i, s := range o.Today {
		sx := colX + float64(i)*146
		vx := text(dc, s.Value, sx, 158, light, 38, textHi, 0, 0)
		text(dc, s.Unit, vx+3, 158, light, 17, textMid, 0, 0)
		text(dc, s.Label, sx, 183, regular, 13.5, textLow, 0, 0)
	}

	label(dc, "ACTIVITÉ · 60 MIN", colX, 232)
	text(dc, o.ActivityNow, right, 232, regular, 13.5, textMid, 1, 0)
	drawActivity(dc, colX, 248, right-colX, 50, o.Activity)

	dc.SetColor(border)
	dc.SetLineWidth(1)
	dc.DrawLine(left, 330, right, 330)
	dc.Stroke()
	label(dc, "SESSIONS", left, 362)
	if len(o.Sessions) == 0 {
		text(dc, "aucune session active", left, 398, regular, 14.5, textLow, 0, 0)
	}
	for i, s := range o.Sessions {
		drawSession(dc, left, right, 398+float64(i)*42, s, o.Pulse)
	}

	const footY = 578
	fx := left
	if len(o.Tools) > 0 {
		fx = text(dc, "outils · 1 h", fx, footY, regular, 13.5, textLow, 0, 0) + 15
		for _, t := range o.Tools {
			fx = text(dc, t.Name, fx, footY, regular, 13.5, textMid, 0, 0) + 5
			fx = text(dc, itoa(float64(t.Count)), fx, footY, medium, 13.5, textHi, 0, 0) + 15
		}
	}
	text(dc, o.Credits, right, footY, regular, 13.5, textLow, 1, 0)

	drawSignature(dc)
	return dc.Image().(*image.RGBA)
}

func label(dc *canvas, s string, x, y float64) {
	text(dc, s, x, y, medium, 11.5, textLow, 0, 2.2)
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

	text(dc, l.Label, cx, cy+r+38, medium, 11.5, textMid, 0.5, 2.2)
	text(dc, l.Reset, cx, cy+r+60, regular, 13.5, textLow, 0.5, 0)
}

func drawActivity(dc *canvas, x, y, w, h float64, values []float64) {
	n := len(values)
	if n == 0 {
		return
	}
	peak := 0.0
	for _, v := range values {
		peak = math.Max(peak, v)
	}
	step := w / float64(n)
	bw := math.Max(1, step-2)
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

func drawSession(dc *canvas, left, right, y float64, s Session, pulse float64) {
	dot, doing := textLow, textLow
	switch s.State {
	case Working:
		dot, doing = green, textMid
	case Asking:
		dot, doing = yellow, yellow
	case Done:
		dot, doing = textMid, textMid
	}
	if s.State == Working {
		dc.SetColor(alpha(green, 0.25*(1-pulse)))
		dc.DrawCircle(left+5, y-5, 5+7*pulse)
		dc.Fill()
	}
	dc.SetColor(dot)
	dc.DrawCircle(left+5, y-5, 5)
	dc.Fill()

	x := text(dc, s.Project, left+22, y, medium, 15.5, textHi, 0, 0)
	text(dc, s.Detail, x+11, y, regular, 13.5, textLow, 0, 0)

	text(dc, s.Doing, 540, y, regular, 14, doing, 0, 0)

	const cx, cw = 708.0, 78.0
	dc.SetLineCapRound()
	dc.SetLineWidth(3.5)
	dc.SetColor(track)
	dc.DrawLine(cx, y-5, cx+cw, y-5)
	dc.Stroke()
	if s.ContextFrac > 0 {
		dc.SetColor(textMid)
		dc.DrawLine(cx, y-5, cx+cw*math.Min(1, s.ContextFrac), y-5)
		dc.Stroke()
	}
	text(dc, s.Context, cx+cw+10, y, regular, 13.5, textMid, 0, 0)
	text(dc, s.Age, right, y, regular, 13.5, textLow, 1, 0)
}
