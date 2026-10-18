package i18n

import (
	"fmt"
	"os"
	"strings"
)

type Plural struct{ One, Other string }

type Texts struct {
	Usage         string
	Fatal         string
	UnknownRender string
	UnknownLang   string

	TokenExpired     string
	RateLimited      string
	RateLimitedRetry string
	ReadCredentials  string
	BadCredentials   string
	NoToken          string
	APICall          string
	ReadResponse     string
	UnexpectedStatus string
	BadResponse      string
	RTKFailed        string
	RTKBadOutput     string

	Demo       string
	Connecting string
	Updated    string
	TooSmall   string
	Keys       string
	PoweredBy  string

	FiveHours   string
	Week        string
	Loading     string
	Unavailable string
	ResetIn     string

	Today     string
	Replies   Plural
	Generated string
	FromCache string

	ByProduct string
	Products  map[string]string

	Activity string
	Quiet    string

	RTK        string
	OnAverage  string
	SavedToday string
	AllTime    string
	DayLetters [7]string

	Sessions    string
	NoSession   string
	Idle        string
	Asking      string
	Interrupted string
	Finished    string
	Thinking    string
	Writing     string
	Using       string
	ToolVerbs   map[string]string

	Tools   string
	Credits string
	Of      string

	Days, Hours, Minutes, Seconds string
	JustNow, Ago                  string

	Decimal     string
	Percent     string
	SymbolFirst bool
	zeroIsOne   bool
}

var T = detect()

func detect() *Texts {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(key); v != "" {
			if strings.HasPrefix(v, "fr") {
				return &fr
			}
			return &en
		}
	}
	return &en
}

func Set(code string) error {
	switch strings.ToLower(code) {
	case "fr":
		T = &fr
	case "en":
		T = &en
	default:
		return fmt.Errorf(T.UnknownLang, code)
	}
	return nil
}

func (t *Texts) Pick(n int, p Plural) string {
	if n == 1 || n == 0 && t.zeroIsOne {
		return p.One
	}
	return p.Other
}
