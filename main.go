package main

import (
	"flag"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/promwish"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wishlist"
	"github.com/gliderlabs/ssh"
	"github.com/maaslalani/confetty/confetti"
	"github.com/maaslalani/confetty/fireworks"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	port        = flag.Int("port", 2222, "port to listen on")
	metricsPort = flag.Int("metrics-port", 9222, "port to listen on")
)

const (
	effectConfetti  = "confetti"
	effectFireworks = "fireworks"
)

func main() {
	flag.Parse()

	go promwish.Listen(fmt.Sprintf("0.0.0.0:%d", *metricsPort))

	cfg := &wishlist.Config{
		Port: int64(*port),
		Factory: func(e wishlist.Endpoint) (*ssh.Server, error) {
			return wish.NewServer(
				wish.WithAddress(e.Address),
				wish.WithMiddleware(
					append(
						e.Middlewares,
						promwish.MiddlewareRegistry(
							prometheus.DefaultRegisterer,
							prometheus.Labels{
								"app": e.Name,
							},
						),
						lm.Middleware(),
						activeterm.Middleware(),
					)...,
				),
			)
		},
		Endpoints: []*wishlist.Endpoint{
			{
				Name: effectConfetti,
				Middlewares: []wish.Middleware{
					bm.Middleware(teaHandler(effectConfetti)),
				},
			},
			{
				Name: effectFireworks,
				Middlewares: []wish.Middleware{
					bm.Middleware(teaHandler(effectFireworks)),
				},
			},
		},
	}

	// start all the servers
	if err := wishlist.Serve(cfg); err != nil {
		log.Fatalln(err)
	}
}

func teaHandler(effect string) func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		var m tea.Model
		switch effect {
		case effectConfetti:
			m = confetti.InitialModel()
		case effectFireworks:
			m = fireworks.InitialModel()
		default:
			log.Fatalf("invalid effect %q", effect)
		}

		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}
