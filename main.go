package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/promwish"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/accesscontrol"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/gliderlabs/ssh"
	"github.com/maaslalani/confetty/confetti"
	"github.com/maaslalani/confetty/fireworks"
)

var port = flag.Int("port", 2222, "port to listen on")
var metricsPort = flag.Int("metrics-port", 9222, "port to listen on")
var effect = flag.String("effect", "confetti", "effect to use [confetti|fireworks]")

func main() {
	flag.Parse()

	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("0.0.0.0:%d", *port)),
		wish.WithHostKeyPath(".ssh/confettysh"),
		wish.WithMiddleware(
			bm.Middleware(teaHandler()),
			lm.Middleware(),
			promwish.Middleware(fmt.Sprintf("0.0.0.0:%d", *metricsPort), *effect),
			accesscontrol.Middleware(),
			activeterm.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Starting SSH server on 0.0.0.0:%d", *port)
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		if err = s.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()
	<-done
	if err := s.Close(); err != nil {
		log.Fatalln(err)
	}
}

func teaHandler() func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		var m tea.Model
		switch *effect {
		case "confetti":
			m = confetti.InitialModel()
		case "fireworks":
			m = fireworks.InitialModel()
		default:
			log.Fatalf("invalid effect %q", *effect)
		}

		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}
