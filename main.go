package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
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
	"github.com/hashicorp/go-multierror"
	"github.com/maaslalani/confetty/confetti"
	"github.com/maaslalani/confetty/fireworks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var port = flag.Int("port", 2222, "port to listen on")
var metricsPort = flag.Int("metrics-port", 9222, "port to listen on")
var effect = flag.String("effect", "confetti", "effect to use [confetti|fireworks]")

func main() {
	flag.Parse()

	http.Handle("/metrics", promhttp.Handler())
	address := fmt.Sprintf("http://0.0.0.0:%d", *metricsPort)
	go http.ListenAndServe(address, nil)
	log.Println("Starting metrics server on", address)

	if err := reverseproxy(&Config{
		Endpoints: []Endpoint{
			{
				Name:        "confetti",
				Address:     fmt.Sprintf("0.0.0.0:%d", -1),
				HostKeyPath: ".ssh/confetti",
				Handler: func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
					return confetti.InitialModel(), []tea.ProgramOption{tea.WithAltScreen()}
				},
			},
			{
				Name:        "fireworks",
				Address:     fmt.Sprintf("0.0.0.0:%d", *port+1),
				HostKeyPath: ".ssh/fireworks",
				Handler: func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
					return fireworks.InitialModel(), []tea.ProgramOption{tea.WithAltScreen()}
				},
			},
		},
	}); err != nil {
		log.Fatal(err)
	}
}

type Endpoint struct {
	Name        string
	Address     string
	HostKeyPath string
	Handler     bm.BubbleTeaHandler
}

type Config struct {
	Endpoints []Endpoint
}

func reverseproxy(config *Config) error {
	var closes []func() error
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	for _, endpoint := range config.Endpoints {
		s, err := wish.NewServer(
			wish.WithAddress(endpoint.Address),
			wish.WithHostKeyPath(endpoint.HostKeyPath),
			wish.WithMiddleware(
				bm.Middleware(endpoint.Handler),
				lm.Middleware(),
				promwish.MiddlewareRegistry(prometheus.DefaultRegisterer, endpoint.Name),
				accesscontrol.Middleware(),
				activeterm.Middleware(),
			),
		)
		if err != nil {
			if cerr := closeAll(closes); cerr != nil {
				return multierror.Append(err, cerr)
			}
			return err
		}
		log.Printf("Starting SSH server for %s on ssh://%s", endpoint.Name, endpoint.Address)
		go s.ListenAndServe()
		closes = append(closes, s.Close)
	}
	<-done
	log.Print("Stopping SSH servers")
	return closeAll(closes)
}

func closeAll(closes []func() error) error {
	var result error
	for _, close := range closes {
		if err := close(); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}
