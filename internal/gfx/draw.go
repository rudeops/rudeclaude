package gfx

import (
	"image/color"
	"math"

	"github.com/fogleman/gg"
)

type Options struct {
	Scale       float64
	Transparent bool
}

type canvas struct {
	*gg.Context
	s float64
}

var (
	bg      = hex(0x181818)
	border  = hex(0x262626)
	track   = hex(0x2A2A2A)
	textHi  = hex(0xF5F5F5)
	textMid = hex(0x9CA3AF)
	textLow = hex(0x6B7280)
	yellow  = hex(0xFEC824)
	orange  = hex(0xFF8C28)
	red     = hex(0xEF4444)
	green   = hex(0x22C55E)
	blue    = hex(0x60A5FA)
	purple  = hex(0xA78BFA)
)

func fillColor(used float64) color.Color {
	switch {
	case used >= 90:
		return red
	case used >= 70:
		return orange
	default:
		return yellow
	}
}

func text(dc *canvas, s string, x, y float64, w weight, size float64, c color.Color, align, tracking float64) float64 {
	if s == "" {
		return x
	}
	dc.SetFontFace(face(w, size*dc.s))
	total := measureTracked(dc, s, tracking)
	x -= total * align
	dc.Push()
	dc.Identity()
	defer dc.Pop()
	dc.SetColor(c)
	if tracking == 0 {
		dc.DrawString(s, x*dc.s, y*dc.s)
		return x + total
	}
	for _, r := range s {
		ch := string(r)
		dc.DrawString(ch, x*dc.s, y*dc.s)
		cw, _ := dc.MeasureString(ch)
		x += cw/dc.s + tracking
	}
	return x - tracking
}

func measure(dc *canvas, s string, w weight, size float64) float64 {
	dc.SetFontFace(face(w, size*dc.s))
	return measureTracked(dc, s, 0)
}

func measureTracked(dc *canvas, s string, tracking float64) float64 {
	sw, _ := dc.MeasureString(s)
	n := float64(len([]rune(s)))
	return sw/dc.s + tracking*math.Max(0, n-1)
}

func itoa(v float64) string {
	n := int(math.Round(v))
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

func hex(v uint32) color.RGBA {
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}
}

func alpha(c color.Color, a float64) color.Color {
	r, g, b, _ := c.RGBA()
	return color.NRGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a * 255)}
}
