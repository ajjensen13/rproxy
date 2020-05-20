package main

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/ajjensen13/config"
	"github.com/ajjensen13/life"
	"github.com/ajjensen13/urlutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ajjensen13/rproxy/internal/rproxy"
)

type Config struct {
	Domains []string `json:"domains"`
	Routes  []*Route `json:"routes"`
	Bucket  string   `json:"bucket"`
}

type RouteType string

const (
	Proxy    RouteType = "proxy"
	Redirect RouteType = "redirect"
)

type Route struct {
	Type RouteType `json:"type"`

	// Pattern is the HTTP pattern that is registered for the associated rule.
	// Pattern is only used for routes of type "proxy" and "redirect".
	Pattern string `json:"pattern"`
	// Rule is the url re-writing rule associated with a pattern.
	// Rule is only used for routes of type "proxy" and "redirect".
	Rule string `json:"rule"`
}

func init() {
	life.OnInit(func(ctx context.Context) error {
		cfg, err := loadConfig()
		if err != nil {
			log.Print(err)
			return err
		}

		err = setupRoutes(cfg)
		if err != nil {
			log.Print(err)
			return err
		}

		log.Printf("main: hardcoded domains: %v", cfg.Domains)
		ids := extractHostsFromRoutes(cfg)
		log.Printf("main: inferred domains: %v", ids)
		domains := append(cfg.Domains, ids...)

		gs, err := storage.NewClient(ctx)
		if err != nil {
			log.Print(err)
			return err
		}

		life.OnDefer(func(_ context.Context) error {
			return gs.Close()
		})

		bucket := gs.Bucket(cfg.Bucket)

		cache := rproxy.NewLayeredCache(
			rproxy.NewMemCache(),
			rproxy.NewGStorageCache(bucket),
		)

		life.OnReady(func(ctx context.Context) error {
			ln := rproxy.NewListener(cache, domains)

			switch err := http.Serve(ln, nil); err {
			case http.ErrServerClosed:
				log.Print("rproxy: server shut down gracefully")
				return nil
			default:
				return fmt.Errorf("rproxy: server shut down upon error: %w", err)
			}
		})

		return nil
	})
}

func main() {
	log.Printf("rproxy: starting up")
	err := life.Start(context.Background(), log.New(log.Writer(), log.Prefix(), log.Flags()))
	if err != nil {
		log.Printf("rproxy: shut down up: %v", err)
		os.Exit(2)
	}
	log.Print("rproxy: shut down up gracefully")
}

func extractHostsFromRoutes(cfg *Config) (hosts []string) {
	for _, route := range cfg.Routes {
		host, ok := hostFromPattern(route.Pattern)
		if ok {
			hosts = append(hosts, host)
		}
	}
	return
}

func setupRoutes(cfg *Config) error {
	for _, route := range cfg.Routes {
		pattern := route.Pattern
		rule := urlutil.Rewriter(route.Rule)
		typ := route.Type

		switch typ {
		case Proxy:
			log.Printf("rproxy: registering %s rule: %s -> %s", typ, pattern, rule)
			http.Handle(pattern, rproxy.NewReverseProxyHandler(rule))
		case Redirect:
			log.Printf("rproxy: registering %s rule: %s -> %s", typ, pattern, rule)
			http.Handle(pattern, rproxy.NewRedirectHandler(rule))
		default:
			return fmt.Errorf("rproxy: error setting up routes: invalid type: %s", typ)
		}
	}
	return nil
}

func hostFromPattern(pattern string) (host string, ok bool) {
	s := strings.SplitN(pattern, "/", 2)
	if len(s) < 2 {
		return "", false
	}

	return s[0], s[0] != ""
}

func loadConfig() (*Config, error) {
	cfg := new(Config)

	err := config.InterfaceJson("rproxy.json", cfg)
	if err != nil {
		return nil, fmt.Errorf("rproxy: error loading config: %w", err)
	}

	log.Printf("rproxy: config loaded: %v", &cfg)
	return cfg, nil
}
