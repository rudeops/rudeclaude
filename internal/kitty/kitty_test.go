package kitty

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"image"
	"io"
	"strings"
	"testing"
)

func TestShowSendsStraightAlpha(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 1))
	copy(img.Pix, []byte{34, 34, 34, 36, 200, 100, 0, 255, 0, 0, 0, 0})
	var out bytes.Buffer
	if err := Show(&out, img, 1, Placement{Col: 1, Row: 1}); err != nil {
		t.Fatal(err)
	}
	s := out.String()
	s = s[strings.Index(s, "\x1b_G"):]
	payload := s[strings.Index(s, ";")+1 : strings.LastIndex(s, "\x1b\\")]
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(zr)
	want := []byte{241, 241, 241, 36, 200, 100, 0, 255, 0, 0, 0, 0}
	if !bytes.Equal(got, want) {
		t.Errorf("pixels envoyés %v, attendu %v", got, want)
	}
	if !bytes.Equal(img.Pix[:4], []byte{34, 34, 34, 36}) {
		t.Errorf("l'image d'origine a été modifiée : %v", img.Pix[:4])
	}
}
