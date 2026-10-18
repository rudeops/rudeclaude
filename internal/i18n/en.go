package i18n

var en = Texts{
	Usage: `Usage: rudeclaude [options]

Options:
  --interval duration  delay between two API calls (1m by default, 30s minimum)
  --render mode        rendering: auto, image (kitty graphics protocol) or text
  --lang language      interface language: fr or en (system language by default)
  --demo               fake data, no network call and no log reading
  --snapshot file      exports a PNG image of the dashboard, then exits
  --version            prints the version, then exits

Keys: r to refresh, q to quit.
`,
	Fatal:         "rudeclaude: %v\n",
	UnknownRender: "unknown rendering: %q (auto, image or text)",
	UnknownLang:   "unknown language: %q (fr or en)",

	TokenExpired:     "token expired: run claude to refresh it",
	RateLimited:      "API rate limited",
	RateLimitedRetry: "API rate limited, retrying in %s",
	ReadCredentials:  "reading Claude Code credentials: %w",
	BadCredentials:   "unreadable Claude Code credentials: %w",
	NoToken:          "no OAuth token: sign in with claude",
	APICall:          "API call: %w",
	ReadResponse:     "reading the response: %w",
	UnexpectedStatus: "unexpected API response: %s",
	BadResponse:      "unreadable response: %w",
	RTKFailed:        "rtk gain: %w",
	RTKBadOutput:     "unreadable rtk gain output: %w",

	Demo:       "demo mode",
	Connecting: "connecting…",
	Updated:    "updated %s ago",
	TooSmall:   "enlarge the window to display rudeclaude · q to quit",
	Keys:       "r refresh · q quit",
	PoweredBy:  "powered by ",

	FiveHours:   "SESSION · 5 H",
	Week:        "WEEK",
	Loading:     "loading…",
	Unavailable: "unavailable",
	ResetIn:     "resets in %s",

	Today:     "TODAY",
	Replies:   Plural{"reply", "replies"},
	Generated: "tokens generated",
	FromCache: "served from cache",

	ByProduct: "WEEK BY PRODUCT",
	Products: map[string]string{
		"claude_code": "Claude Code",
		"chat":        "Chat",
		"cowork":      "Cowork",
		"other":       "Other",
	},

	Activity: "ACTIVITY · 60 MIN",
	Quiet:    "quiet",

	RTK:        "RTK · 7 DAYS",
	OnAverage:  "on average",
	SavedToday: "saved today",
	AllTime:    "all time",
	DayLetters: [7]string{"S", "M", "T", "W", "T", "F", "S"},

	Sessions:    "SESSIONS",
	NoSession:   "no active session",
	Idle:        "idle",
	Asking:      "asking you a question",
	Interrupted: "interrupted",
	Finished:    "done",
	Thinking:    "thinking…",
	Writing:     "writing…",
	Using:       "using %s",
	ToolVerbs: map[string]string{
		"Bash":         "running a command",
		"Read":         "reading a file",
		"Edit":         "editing a file",
		"MultiEdit":    "editing a file",
		"Write":        "writing a file",
		"NotebookEdit": "editing a notebook",
		"Grep":         "searching the code",
		"Glob":         "looking for files",
		"WebFetch":     "browsing the web",
		"WebSearch":    "searching the web",
		"Agent":        "delegating to an agent",
		"Task":         "delegating to an agent",
		"TodoWrite":    "planning",
		"Skill":        "loading a skill",
	},

	Tools:   "tools · 1h",
	Credits: "extra credits · %s",
	Of:      " of %s",

	Days:    "%dd %dh",
	Hours:   "%dh %02dm",
	Minutes: "%dm",
	Seconds: "%ds",
	JustNow: "just now",
	Ago:     "%s ago",

	Decimal:     ".",
	Percent:     "%",
	SymbolFirst: true,
}
