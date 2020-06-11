package main

import (
	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/ajjensen13/config"
	"github.com/ajjensen13/gke"
	"github.com/ajjensen13/urlutil"
	"golang.org/x/crypto/acme/autocert"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ajjensen13/rproxy/internal/rproxy"
)

type Config struct {
	Routes []*Route `json:"routes"`
	Bucket string   `json:"bucket"`
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

func main() {
	startupCtx, startupCancel := context.WithTimeout(context.Background(), time.Second*15)
	defer startupCancel()

	lg, cleanUpLogger := newLogger(startupCtx)
	defer cleanUpLogger()

	ln, cleanUpListener := newListener(startupCtx, lg)
	defer cleanUpListener()

	server := newServer(startupCtx, lg)

	switch err := server.Serve(ln); err {
	case http.ErrServerClosed:
		lg.Noticef("shut down up gracefully")
	default:
		panic(lg.WarnErr(fmt.Errorf("shut down: %w", err)))
	}
}

func provideLogClient(ctx context.Context) (*gke.LogClient, func()) {
	lc, err := gke.NewLogClient(ctx)
	if err != nil {
		panic(err)
	}
	return lc, func() { _ = lc.Close() }
}

func provideLogger(lgc *gke.LogClient, lgId string) (*gke.Logger, func()) {
	l := lgc.Logger(lgId)
	return l, func() { _ = l.Flush() }
}

func provideConfig(lg *gke.Logger) *Config {
	var result Config

	err := config.InterfaceYaml("rproxy.yaml", &result)
	if err != nil {
		panic(lg.ErrorErr(fmt.Errorf("rproxy: error loading config: %w", err)))
	}

	lg.Infof("provided config: %v", &result)
	return &result
}

func provideDomains(lg *gke.Logger, cfg *Config) (domains []string) {
	for _, route := range cfg.Routes {
		host, ok := hostFromPattern(route.Pattern)
		if ok {
			domains = append(domains, host)
		}
	}
	lg.Infof("provided domains: %v", domains)
	return
}

func provideStorageClient(ctx context.Context, lg *gke.Logger) (*storage.Client, func()) {
	result, err := storage.NewClient(ctx)
	if err != nil {
		panic(lg.ErrorErr(err))
	}
	lg.Infof("provided storage client: %v", result)
	return result, func() { _ = result.Close() }
}

func provideBucketHandle(lg *gke.Logger, cfg *Config, gs *storage.Client) *storage.BucketHandle {
	result := gs.Bucket(cfg.Bucket)
	lg.Infof("provided bucket handle: %v", result)
	return result
}

func provideAutocertCache(lg *gke.Logger, bucket *storage.BucketHandle) autocert.Cache {
	result := rproxy.NewLayeredCache(
		lg,
		rproxy.NewMemCache(),
		rproxy.NewGStorageCache(lg, bucket),
	)
	lg.Infof("provided autocert cache: %v", result)
	return result
}

func provideListener(lg *gke.Logger, cache autocert.Cache, domains []string) net.Listener {
	result := rproxy.NewListener(cache, domains)
	lg.Infof("provided listener: %v", result)
	return result
}

func hostFromPattern(pattern string) (host string, ok bool) {
	s := strings.SplitN(pattern, "/", 2)
	if len(s) < 2 {
		return "", false
	}

	return s[0], s[0] != ""
}

func provideHandler(lg *gke.Logger, cfg *Config) http.Handler {
	for _, route := range cfg.Routes {
		pattern := route.Pattern
		rule := urlutil.Rewriter(route.Rule)
		typ := route.Type

		switch typ {
		case Proxy:
			lg.Infof("registering %s rule: %s -> %s", typ, pattern, rule)
			http.Handle(pattern, rproxy.NewReverseProxyHandler(lg, rule))
		case Redirect:
			lg.Infof("registering %s rule: %s -> %s", typ, pattern, rule)
			http.Handle(pattern, rproxy.NewRedirectHandler(lg, rule))
		default:
			panic(lg.ErrorErr(fmt.Errorf("error setting up routes: invalid type: %s", typ)))
		}
	}
	lg.Infof("provided handler: %v", http.DefaultServeMux)
	return http.DefaultServeMux
}

func provideServer(lg *gke.Logger, handler http.Handler, errorLog *log.Logger) *http.Server {
	result := http.Server{
		Handler:  handler,
		ErrorLog: errorLog,
		BaseContext: func(_ net.Listener) context.Context {
			ctx, _ := gke.Alive()
			return ctx
		},
		ReadHeaderTimeout: time.Second * 3,
		ReadTimeout:       time.Second * 15,
		WriteTimeout:      time.Second * 15,
	}

	lg.Infof("provided server: %v", &result)
	return &result
}

func provideErrorLogger(lg *gke.Logger) *log.Logger {
	return lg.StandardLogger(logging.Error)
}
