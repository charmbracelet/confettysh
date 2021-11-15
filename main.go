package main

import (
	"fmt"
	"log"

	"github.com/caarlos0/env/v6"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
	"github.com/maaslalani/confetty/confetti"
	"github.com/maaslalani/confetty/fireworks"
	"github.com/muesli/termenv"
)

type Config struct {
	Host   string `env:"CONFETTYSH_HOST" envDefault:"127.0.0.1"`
	Port   int    `env:"CONFETTYSH_PORT" envDefault:"2222"`
	Effect string `env:"CONFETTYSH_EFFECT" envDefault:"confetti"`
}

func main() {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalln(err)
	}

	// force colors as we might start it from systemd which has no interactive term and no colors
	lipgloss.SetColorProfile(termenv.ANSI256)

	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		wish.WithHostKeyPath(".ssh/confettysh"),
		wish.WithMiddleware(
			bm.Middleware(teaHandler(cfg)),
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Starting SSH server on %s:%d", cfg.Host, cfg.Port)
	err = s.ListenAndServe()
	if err != nil {
		log.Fatalln(err)
	}
}

func teaHandler(cfg Config) func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		_, _, active := s.Pty()
		if !active {
			fmt.Println("no active terminal, skipping")
			return nil, nil
		}

		var m tea.Model
		switch cfg.Effect {
		case "confetti":
			m = confetti.InitialModel()
		case "fireworks":
			m = fireworks.InitialModel()
		default:
			log.Fatalf("invalid effect %q", cfg.Effect)
		}

		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}
