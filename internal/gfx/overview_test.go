package gfx

import (
	"testing"

	"github.com/fogleman/gg"
)

func TestHatch(t *testing.T) {
	const size = 40
	dc := &canvas{gg.NewContext(size, size), 1}
	hatch(dc, 10, 10, 20, 20, 1.5, yellow)
	img := dc.Image()

	alphas := map[uint32]bool{}
	for py := range size {
		for px := range size {
			_, _, _, a := img.At(px, py).RGBA()
			inside := px >= 10 && px < 30 && py >= 10 && py < 30
			if !inside && a != 0 {
				t.Fatalf("pixel (%d,%d) hors de la barre dessiné", px, py)
			}
			if inside {
				alphas[a] = true
			}
		}
	}
	if len(alphas) < 2 {
		t.Errorf("%d niveau(x) d'opacité dans la barre, attendu des rayures sur un fond translucide", len(alphas))
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Errorf("le fond devrait rester transparent, alpha = %d", a)
	}
}
