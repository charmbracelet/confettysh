package main

import (
	"fmt"
	"log"
	"net"
	"runtime/debug"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/promwish"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wishlist"
	"github.com/maaslalani/confetty/confetti"
	"github.com/maaslalani/confetty/fireworks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
)

// nolint: gomnd
var (
	listen      = pflag.String("listen", "localhost", "address to listen to")
	port        = pflag.Int("port", 2222, "port to listen on")
	metricsPort = pflag.Int("metrics-port", 9222, "port to listen on")
)

const (
	effectConfetti  = "confetti"
	effectFireworks = "fireworks"
)

func main() {
	pflag.Parse()

	version := "devel"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
		version = info.Main.Version
	}
	log.Printf("Running confettysh %s", version)

	go promwish.Listen(net.JoinHostPort(*listen, strconv.Itoa(*metricsPort)))

	cfg := &wishlist.Config{
		Listen: *listen,
		Port:   int64(*port),
		Factory: func(e wishlist.Endpoint) (*ssh.Server, error) {
			return wish.NewServer(
				wish.WithAddress(e.Address),
				wish.WithHostKeyPath(fmt.Sprintf(".ssh/%s", strings.ToLower(e.Name))),
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
				Name: "Confetti",
				Middlewares: []wish.Middleware{
					bm.Middleware(teaHandler(effectConfetti)),
				},
			},
			{
				Name: "Fireworks",
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
		}

		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}
