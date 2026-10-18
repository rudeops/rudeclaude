package i18n

import (
	"maps"
	"reflect"
	"slices"
	"testing"
)

func TestTablesComplete(t *testing.T) {
	for name, table := range map[string]*Texts{"fr": &fr, "en": &en} {
		v := reflect.ValueOf(table).Elem()
		for i := range v.NumField() {
			field, f := v.Type().Field(i).Name, v.Field(i)
			switch f.Kind() {
			case reflect.String:
				if f.String() == "" {
					t.Errorf("%s.%s : texte vide", name, field)
				}
			case reflect.Struct:
				for j := range f.NumField() {
					if f.Field(j).String() == "" {
						t.Errorf("%s.%s.%s : texte vide", name, field, f.Type().Field(j).Name)
					}
				}
			case reflect.Array:
				for j := range f.Len() {
					if f.Index(j).String() == "" {
						t.Errorf("%s.%s[%d] : texte vide", name, field, j)
					}
				}
			}
		}
	}
	for _, pair := range [][2]map[string]string{{fr.Products, en.Products}, {fr.ToolVerbs, en.ToolVerbs}} {
		a, b := slices.Sorted(maps.Keys(pair[0])), slices.Sorted(maps.Keys(pair[1]))
		if !slices.Equal(a, b) {
			t.Errorf("clés différentes : fr %v, en %v", a, b)
		}
	}
}

func TestDetect(t *testing.T) {
	cases := []struct {
		all, messages, lang string
		want                *Texts
	}{
		{"", "", "fr_FR.UTF-8", &fr},
		{"", "", "en_US.UTF-8", &en},
		{"", "fr_BE.UTF-8", "en_US.UTF-8", &fr},
		{"C", "", "fr_FR.UTF-8", &en},
		{"fr_CA.UTF-8", "", "", &fr},
		{"", "", "", &en},
	}
	for _, c := range cases {
		t.Setenv("LC_ALL", c.all)
		t.Setenv("LC_MESSAGES", c.messages)
		t.Setenv("LANG", c.lang)
		if got := detect(); got != c.want {
			t.Errorf("LC_ALL=%q LC_MESSAGES=%q LANG=%q : mauvaise langue", c.all, c.messages, c.lang)
		}
	}
}

func TestSet(t *testing.T) {
	defer func(prev *Texts) { T = prev }(T)
	if err := Set("EN"); err != nil || T != &en {
		t.Errorf("Set(EN) : %v", err)
	}
	if err := Set("fr"); err != nil || T != &fr {
		t.Errorf("Set(fr) : %v", err)
	}
	if err := Set("de"); err == nil || T != &fr {
		t.Errorf("Set(de) : attendu une erreur sans changer de langue")
	}
}

func TestPick(t *testing.T) {
	for _, c := range []struct {
		t    *Texts
		n    int
		want string
	}{
		{&fr, 0, "réponse"}, {&fr, 1, "réponse"}, {&fr, 2, "réponses"},
		{&en, 0, "replies"}, {&en, 1, "reply"}, {&en, 2, "replies"},
	} {
		if got := c.t.Pick(c.n, c.t.Replies); got != c.want {
			t.Errorf("Pick(%d) = %q, attendu %q", c.n, got, c.want)
		}
	}
}
