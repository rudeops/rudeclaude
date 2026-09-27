package gfx

import (
	"embed"
	"fmt"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

//go:embed fonts/*.otf
var fontFiles embed.FS

type weight string

const (
	light   weight = "Light"
	regular weight = "Regular"
	medium  weight = "Medium"
)

type faceKey struct {
	w    weight
	size float64
}

var (
	fontMu sync.Mutex
	fonts  = map[weight]*opentype.Font{}
	faces  = map[faceKey]font.Face{}
)

func face(w weight, size float64) font.Face {
	fontMu.Lock()
	defer fontMu.Unlock()
	key := faceKey{w, size}
	if f, ok := faces[key]; ok {
		return f
	}
	f, ok := fonts[w]
	if !ok {
		raw, err := fontFiles.ReadFile(fmt.Sprintf("fonts/Inter-%s.otf", w))
		if err != nil {
			panic(err)
		}
		if f, err = opentype.Parse(raw); err != nil {
			panic(err)
		}
		fonts[w] = f
	}
	ff, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		panic(err)
	}
	faces[key] = ff
	return ff
}
