package i18n

var fr = Texts{
	Usage: `Utilisation : rudeclaude [options]

Options :
  --interval durée     délai entre deux appels à l'API (1 min par défaut, 30 s minimum)
  --render mode        rendu : auto, image (protocole graphique de kitty) ou text
  --lang langue        langue de l'interface : fr ou en (celle du système par défaut)
  --demo               données fictives, sans appel réseau ni lecture des journaux
  --snapshot fichier   exporte une image PNG du tableau de bord, puis quitte
  --version            affiche la version, puis quitte

Touches : r pour rafraîchir, q pour quitter.
`,
	Fatal:         "rudeclaude : %v\n",
	UnknownRender: "rendu inconnu : %q (auto, image ou text)",
	UnknownLang:   "langue inconnue : %q (fr ou en)",

	TokenExpired:     "token expiré : lance claude pour le rafraîchir",
	RateLimited:      "API saturée",
	RateLimitedRetry: "API saturée, nouvel essai dans %s",
	ReadCredentials:  "lecture des identifiants Claude Code : %w",
	BadCredentials:   "identifiants Claude Code illisibles : %w",
	NoToken:          "aucun token OAuth : connecte-toi avec claude",
	APICall:          "appel API : %w",
	ReadResponse:     "lecture de la réponse : %w",
	UnexpectedStatus: "réponse inattendue de l'API : %s",
	BadResponse:      "réponse illisible : %w",
	RTKFailed:        "rtk gain : %w",
	RTKBadOutput:     "rtk gain illisible : %w",

	Demo:       "mode démo",
	Connecting: "connexion…",
	Updated:    "mis à jour il y a %s",
	TooSmall:   "agrandis la fenêtre pour afficher rudeclaude · q pour quitter",
	Keys:       "r rafraîchir · q quitter",
	PoweredBy:  "propulsé par ",

	FiveHours:   "SESSION · 5 H",
	Week:        "SEMAINE",
	Loading:     "chargement…",
	Unavailable: "indisponible",
	ResetIn:     "reset dans %s",

	Today:     "AUJOURD'HUI",
	Replies:   Plural{"réponse", "réponses"},
	Generated: "tokens générés",
	FromCache: "servis par le cache",

	ByProduct: "SEMAINE PAR PRODUIT",
	Products: map[string]string{
		"claude_code": "Claude Code",
		"chat":        "Chat",
		"cowork":      "Cowork",
		"other":       "Autre",
	},

	Activity: "ACTIVITÉ · 60 MIN",
	Quiet:    "au calme",

	RTK:        "RTK · 7 JOURS",
	OnAverage:  "en moyenne",
	SavedToday: "économisés aujourd'hui",
	AllTime:    "au total",
	DayLetters: [7]string{"D", "L", "M", "M", "J", "V", "S"},

	Sessions:    "SESSIONS",
	NoSession:   "aucune session active",
	Idle:        "inactive",
	Asking:      "te pose une question",
	Interrupted: "interrompu",
	Finished:    "terminé",
	Thinking:    "réfléchit…",
	Writing:     "rédige…",
	Using:       "utilise %s",
	ToolVerbs: map[string]string{
		"Bash":         "exécute une commande",
		"Read":         "lit un fichier",
		"Edit":         "modifie un fichier",
		"MultiEdit":    "modifie un fichier",
		"Write":        "écrit un fichier",
		"NotebookEdit": "modifie un notebook",
		"Grep":         "cherche dans le code",
		"Glob":         "cherche des fichiers",
		"WebFetch":     "consulte le web",
		"WebSearch":    "cherche sur le web",
		"Agent":        "délègue à un agent",
		"Task":         "délègue à un agent",
		"TodoWrite":    "planifie",
		"Skill":        "charge une compétence",
	},

	Tools:   "outils · 1 h",
	Credits: "crédits extra · %s",
	Of:      " sur %s",

	Days:    "%d j %d h",
	Hours:   "%d h %02d",
	Minutes: "%d min",
	Seconds: "%d s",
	JustNow: "à l'instant",
	Ago:     "il y a %s",

	Decimal:   ",",
	Percent:   " %",
	zeroIsOne: true,
}
