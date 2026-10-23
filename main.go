package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rudeops/rudeclaude/internal/i18n"
	"github.com/rudeops/rudeclaude/internal/ui"
	"github.com/rudeops/rudeclaude/internal/usage"
)

const version = "0.4.0"

func main() {
	flag.Usage = func() { fmt.Fprint(flag.CommandLine.Output(), i18n.T.Usage) }
	interval := flag.Duration("interval", time.Minute, "")
	demo := flag.Bool("demo", false, "")
	render := flag.String("render", "auto", "")
	snapshot := flag.String("snapshot", "", "")
	showVersion := flag.Bool("version", false, "")
	flag.Func("lang", "", i18n.Set)
	flag.Parse()

	if *showVersion {
		fmt.Println("rudeclaude", version)
		return
	}
	usage.UserAgent = "rudeclaude/" + version
	ui.Version = version

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
		exit(fmt.Errorf(i18n.T.UnknownRender, *render))
	}
}

func runText(interval time.Duration, demo bool) error {
	_, err := tea.NewProgram(ui.NewText(interval, demo), tea.WithAltScreen()).Run()
	return err
}

func exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, i18n.T.Fatal, err)
		os.Exit(1)
	}
}
