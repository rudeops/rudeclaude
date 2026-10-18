package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestTextViewFitsWindow(t *testing.T) {
	for _, lang := range []string{"fr", "en"} {
		inLang(t, lang)
		for _, alert := range []error{nil, errors.New(strings.Repeat("message d'erreur très long ", 8))} {
			c := newCore(time.Minute, true)
			c.now = time.Date(2026, 10, 18, 15, 20, 36, 0, time.UTC)
			c.refresh()
			c.err = alert
			for _, cols := range []int{1, 10, 30, 49, 50, 51, 52, 60, 63, 64, 71, 72, 79, 80, 89, 90, 100, 124, 160, 300} {
				for _, rows := range []int{1, 5, 8, 10, 12, 15, 18, 20, 24, 30, 40, 60, 100} {
					view := textModel{core: c, width: cols, height: rows}.View()
					lines := strings.Split(view, "\n")
					if len(lines) > rows {
						t.Errorf("%s %dx%d : %d lignes", lang, cols, rows, len(lines))
					}
					for _, l := range lines {
						if w := lipgloss.Width(l); w > cols {
							t.Errorf("%s %dx%d : ligne de %d colonnes", lang, cols, rows, w)
							break
						}
					}
				}
			}
		}
	}
}

func TestTextViewKeepsEssentials(t *testing.T) {
	inLang(t, "fr")
	c := newCore(time.Minute, true)
	c.now = time.Date(2026, 10, 18, 15, 20, 36, 0, time.UTC)
	c.refresh()
	for _, size := range [][2]int{{80, 24}, {60, 20}, {52, 14}} {
		view := textModel{core: c, width: size[0], height: size[1]}.View()
		for _, want := range []string{"rudeclaude", "SESSION · 5 H", "SEMAINE", "SESSIONS", "api-paiement"} {
			if !strings.Contains(view, want) {
				t.Errorf("%dx%d : %q absent", size[0], size[1], want)
			}
		}
	}
}
