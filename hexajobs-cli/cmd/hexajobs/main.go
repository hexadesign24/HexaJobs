package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"hexajobs.dev/hexajobs-cli/internal/core"
	"hexajobs.dev/hexajobs-cli/internal/models"
	"hexajobs.dev/hexajobs-cli/internal/ui"
	"hexajobs.dev/hexajobs-cli/internal/ui/views"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "hexajobs:", err)
		os.Exit(1)
	}
}

func run() error {
	demo := flag.Bool("demo", false, "try the interface with offline example listings")
	version := flag.Bool("version", false, "print version")
	config := flag.String("config", "", "engine configuration path (default ~/.config/hexajobs/config.json)")
	flag.Parse()
	if *version {
		fmt.Println("hexajobs.dev " + views.AppVersion)
		return nil
	}
	// The TUI needs a real terminal. Without one Bubble Tea fails with
	// "could not open a new TTY", so report it in plain language instead.
	// (/dev/null is a char device too, so probe /dev/tty directly.)
	if tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err != nil {
		return errors.New("need an interactive terminal (no /dev/tty): run inside a terminal emulator, not via pipe/redirect; --version works anywhere")
	} else {
		_ = tty.Close()
	}
	var engine models.EngineContract
	options := []ui.Option{ui.WithSponsorURL(os.Getenv("HEXAJOBS_SPONSOR_URL"))}
	if *demo {
		engine = ui.DemoEngine{}
		options = append(options, ui.WithDemo())
	} else {
		options = append(options, ui.WithServices(ui.Services{Bootstrap: func() (models.EngineContract, error) {
			cfg, err := core.LoadConfig(*config)
			if err != nil {
				return nil, err
			}
			return core.NewEngine(cfg, core.Options{})
		}}))
	}
	// Configuration, engine creation, explicit purge, and UI preferences are all
	// loaded by the model's Init command, outside the rendering/event loop.
	model := ui.NewModel(engine, options...)
	defer model.Close()
	final, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithFPS(30)).Run()
	if err != nil {
		return err
	}
	if result, ok := final.(ui.Model); ok {
		return result.FatalErr
	}
	return nil
}
