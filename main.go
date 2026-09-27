package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rudeops/rudeclaude/internal/ui"
	"github.com/rudeops/rudeclaude/internal/usage"
)

const version = "0.1.0"

func main() {
	interval := flag.Duration("interval", time.Minute, "délai entre deux appels à l'API (minimum 30s)")
	demo := flag.Bool("demo", false, "valeurs fictives, sans appeler l'API")
	render := flag.String("render", "auto", "rendu : auto, image (protocole graphique de kitty) ou text")
	snapshot := flag.String("snapshot", "", "écrit une image PNG du tableau de bord dans ce fichier, puis quitte")
	showVersion := flag.Bool("version", false, "affiche la version, puis quitte")
	flag.Parse()

	if *showVersion {
		fmt.Println("rudeclaude", version)
		return
	}
	usage.UserAgent = "rudeclaude/" + version

	*interval = max(*interval, 30*time.Second)

	if *snapshot != "" {
		exit(ui.Snapshot(*snapshot, *demo))
		return
	}

	switch *render {
	case "image":
		exit(ui.RunImage(*interval, *demo))
	case "text":
		exit(runText(*interval, *demo))
	case "auto":
		if ui.GraphicsSupported() {
			exit(ui.RunImage(*interval, *demo))
		} else {
			exit(runText(*interval, *demo))
		}
	default:
		exit(fmt.Errorf("rendu inconnu : %q (auto, image ou text)", *render))
	}
}

func runText(interval time.Duration, demo bool) error {
	_, err := tea.NewProgram(ui.NewText(interval, demo), tea.WithAltScreen()).Run()
	return err
}

func exit(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "rudeclaude :", err)
		os.Exit(1)
	}
}
