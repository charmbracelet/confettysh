package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		Listen:  "0.0.0.0",
		Port:    *port,
		Factory: factory,
		Endpoints: []Endpoint{
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

type Endpoint struct {
	Name    string
	Address string
	Handler bm.BubbleTeaHandler
}

type Config struct {
	Listen    string
	Port      int
	Endpoints []Endpoint
	Factory   func(Endpoint) (*ssh.Server, error)
}

func reverseproxy(config *Config) error {
	var closes []func() error
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	for i := range config.Endpoints {
		if config.Endpoints[i].Address == "" {
			config.Endpoints[i].Address = fmt.Sprintf("%s:%d", config.Listen, config.Port+1+i)
		}
	}
	config.Endpoints = append([]Endpoint{
		{
			Name:    "listing",
			Address: fmt.Sprintf("%s:%d", config.Listen, config.Port),
			Handler: func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
				return newListing(config.Endpoints), []tea.ProgramOption{tea.WithAltScreen()}
			},
		},
	}, config.Endpoints...)
	for _, endpoint := range config.Endpoints {
		s, err := config.Factory(endpoint)
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

func factory(endpoint Endpoint) (*ssh.Server, error) {
	return wish.NewServer(
		wish.WithAddress(endpoint.Address),
		wish.WithMiddleware(
			bm.Middleware(endpoint.Handler),
			lm.Middleware(),
			promwish.MiddlewareRegistry(prometheus.DefaultRegisterer, endpoint.Name),
			accesscontrol.Middleware(),
			activeterm.Middleware(),
		),
	)
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

func newListing(endpoints []Endpoint) tea.Model {
	var items []list.Item
	for _, endpoint := range endpoints {
		items = append(items, endpoint)
	}
	l := list.NewModel(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Directory Listing"
	return model{l}
}

type model struct {
	list list.Model
}

func (i Endpoint) Title() string       { return i.Name }
func (i Endpoint) Description() string { return fmt.Sprintf("ssh://%s", i.Address) }
func (i Endpoint) FilterValue() string { return i.Name }

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, nil
		}
	case tea.WindowSizeMsg:
		top, right, bottom, left := docStyle.GetMargin()
		m.list.SetSize(msg.Width-left-right, msg.Height-top-bottom)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}
