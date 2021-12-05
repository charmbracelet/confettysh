package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gliderlabs/ssh"
	"github.com/maaslalani/confetty/confetti"
	"github.com/maaslalani/confetty/fireworks"
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
		Listen:  "0.0.0.0",
		Port:    *port,
		Factory: factory,
		Endpoints: []*Endpoint{
			{
				Name: "confetti",
				Handler: func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
					return confetti.InitialModel(), []tea.ProgramOption{tea.WithAltScreen()}
				},
			},
			{
				Name: "fireworks",
				Handler: func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
					return fireworks.InitialModel(), []tea.ProgramOption{tea.WithAltScreen()}
				},
			},
		},
	}); err != nil {
		log.Fatal(err)
	}
}
