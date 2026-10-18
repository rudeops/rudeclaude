package ui

import (
	"testing"
	"time"

	"github.com/rudeops/rudeclaude/internal/i18n"
	"github.com/rudeops/rudeclaude/internal/usage"
)

func inLang(t *testing.T, code string) {
	t.Helper()
	prev := i18n.T
	if err := i18n.Set(code); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { i18n.T = prev })
}

func TestFormats(t *testing.T) {
	euros := usage.Money{AmountMinor: 350, Currency: "EUR", Exponent: 2}
	francs := usage.Money{AmountMinor: 1200, Currency: "CHF", Exponent: 2}
	cases := []struct {
		lang string
		got  func() string
		want string
	}{
		{"fr", func() string { return formatDuration(95*time.Hour + 30*time.Minute) }, "3 j 23 h"},
		{"en", func() string { return formatDuration(95*time.Hour + 30*time.Minute) }, "3d 23h"},
		{"fr", func() string { return formatDuration(2*time.Hour + 5*time.Minute) }, "2 h 05"},
		{"en", func() string { return formatDuration(2*time.Hour + 5*time.Minute) }, "2h 05m"},
		{"fr", func() string { return formatDuration(12 * time.Minute) }, "12 min"},
		{"en", func() string { return formatDuration(12 * time.Minute) }, "12m"},
		{"fr", func() string { return formatDuration(-time.Second) }, "0 s"},
		{"en", func() string { return formatDuration(45 * time.Second) }, "45s"},
		{"fr", func() string { return formatAge(10 * time.Second) }, "à l'instant"},
		{"en", func() string { return formatAge(10 * time.Second) }, "just now"},
		{"fr", func() string { return formatAge(6 * time.Minute) }, "il y a 6 min"},
		{"en", func() string { return formatAge(6 * time.Minute) }, "6m ago"},
		{"fr", func() string { return formatTokens(6_040_000) }, "6,0M"},
		{"en", func() string { return formatTokens(6_040_000) }, "6.0M"},
		{"en", func() string { return formatTokens(186_000) }, "186k"},
		{"en", func() string { return formatTokens(999) }, "999"},
		{"fr", func() string { return formatMoney(euros) }, "3,50 €"},
		{"en", func() string { return formatMoney(euros) }, "€3.50"},
		{"fr", func() string { return formatMoney(francs) }, "12,00 CHF"},
		{"en", func() string { return formatMoney(francs) }, "12.00 CHF"},
		{"fr", func() string { return percent("95") }, "95 %"},
		{"en", func() string { return percent("95") }, "95%"},
	}
	for _, c := range cases {
		inLang(t, c.lang)
		if got := c.got(); got != c.want {
			t.Errorf("%s : %q, attendu %q", c.lang, got, c.want)
		}
	}
}

func TestErrorsFollowLanguage(t *testing.T) {
	inLang(t, "en")
	if got := usage.ErrTokenExpired.Error(); got != "token expired: run claude to refresh it" {
		t.Errorf("en : %q", got)
	}
	inLang(t, "fr")
	if got := usage.ErrTokenExpired.Error(); got != "token expiré : lance claude pour le rafraîchir" {
		t.Errorf("fr : %q", got)
	}
}
