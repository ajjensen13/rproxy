// +build wireinject

/*
Copyright © 2020 A. Jensen <jensen.aaro@gmail.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/ajjensen13/config"
	"github.com/ajjensen13/gke"
	"github.com/ajjensen13/urlutil"
	"github.com/google/wire"
	"golang.org/x/crypto/acme/autocert"
	"net"
	"net/http"
	"os"
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

const (
	shutdownGracePeriod = time.Second * 15
)

func main() {
	lg, cleanUpLogger, err := newLogger(context.Background())
	if err != nil {
		panic(err)
	}
	defer cleanUpLogger()

	logStartingInfo(lg)

	ctx, _ := gke.Alive()
	shutdownCtx := func() context.Context {
		result, cancel := context.WithCancel(context.Background())
		go func() {
			defer cancel()

			<-ctx.Done()
			lg.Infof("alive context has been cancelled. beginning %v shutdown grace period", shutdownGracePeriod)

			<-time.After(shutdownGracePeriod)
			lg.Infof("shutdown grace period has elapsed.", shutdownGracePeriod)
		}()
		return result
	}()
	err = listenAndServe(ctx, lg)
	<-shutdownCtx.Done() // Wait for shutdownGracePeriod to elapse
	lg.Noticef("shut down complete.")

	if err != nil {
		panic(err)
	}
}

func logStartingInfo(lg gke.Logger) {
	lg.Noticef("starting up.")
	lg.Infof("env: %v", os.Environ())
	gce := gke.OnGCE()
	lg.Infof("OnGCE() = %v", gce)
	if gce {
		lg.Infof("gke.ProjectID() = %v", gke.ProjectID())
		lg.Infof("gke.InstanceName() = %v", gke.InstanceName())
	}
}

func listenAndServe(ctx context.Context, lg gke.Logger) error {
	ln, cleanUpListener, err := newListener(ctx, lg)
	if err != nil {
		panic(err)
	}
	defer cleanUpListener()

	server, cleanupServer, err := newServer(ctx, lg)
	if err != nil {
		panic(err)
	}
	defer cleanupServer()

	go func() {
		<-ctx.Done()
		lg.Infof("shutting down the server")
		_ = server.Shutdown(context.Background())
		lg.Infof("server shut down")
	}()

	lg.Infof("serving")
	switch err := server.Serve(ln); err {
	case http.ErrServerClosed:
		lg.Infof("shut down up gracefully")
		return nil
	default:
		return lg.WarnErr(fmt.Errorf("shut down: %w", err))
	}
}

func provideConfig(lg gke.Logger) *Config {
	var result Config

	err := config.InterfaceYaml("rproxy.yaml", &result)
	if err != nil {
		panic(lg.ErrorErr(fmt.Errorf("rproxy: error loading config: %w", err)))
	}

	lg.Infof("provided config: %#v", &result)
	return &result
}

func provideDomains(lg gke.Logger, cfg *Config) (domains []string) {
	for _, route := range cfg.Routes {
		host, ok := hostFromPattern(route.Pattern)
		if ok {
			domains = append(domains, host)
		}
	}
	lg.Infof("provided domains: %v", domains)
	return
}

func provideBucketHandle(lg gke.Logger, cfg *Config, gs gke.StorageClient) *storage.BucketHandle {
	result := gs.Bucket(cfg.Bucket)
	lg.Infof("provided bucket handle: %v", cfg.Bucket)
	return result
}

func provideAutocertCache(lg gke.Logger, bucket *storage.BucketHandle) autocert.Cache {
	result := rproxy.NewLayeredCache(
		lg,
		rproxy.NewMemCache(),
		rproxy.NewGStorageCache(lg, bucket),
	)
	lg.Infof("provided autocert cache")
	return result
}

func provideListener(lg gke.Logger, cache autocert.Cache, domains []string) net.Listener {
	result := rproxy.NewListener(cache, domains)
	lg.Infof("provided listener: %v", result.Addr())
	return result
}

func hostFromPattern(pattern string) (host string, ok bool) {
	s := strings.SplitN(pattern, "/", 2)
	if len(s) < 2 {
		return "", false
	}

	return s[0], s[0] != ""
}

func provideHandler(lg gke.Logger, cfg *Config) http.Handler {
	for _, route := range cfg.Routes {
		pattern := route.Pattern
		rule := urlutil.Rewriter(route.Rule)
		typ := route.Type

		switch typ {
		case Proxy:
			http.Handle(pattern, rproxy.NewReverseProxyHandler(lg, rule))
		case Redirect:
			http.Handle(pattern, rproxy.NewRedirectHandler(lg, rule))
		default:
			panic(lg.ErrorErr(fmt.Errorf("error setting up routes: invalid type: %s", typ)))
		}
	}
	lg.Infof("provided handler: %v", cfg.Routes)
	return http.DefaultServeMux
}

func newLogger(ctx context.Context) (gke.Logger, func(), error) {
	panic(wire.Build(gke.NewLogClient, gke.NewLogger))
}

func newListener(ctx context.Context, l gke.Logger) (net.Listener, func(), error) {
	panic(wire.Build(provideListener, provideAutocertCache, provideBucketHandle, gke.NewStorageClient, provideDomains, provideConfig))
}

func newServer(ctx context.Context, l gke.Logger) (*http.Server, func(), error) {
	panic(wire.Build(gke.NewServer, provideHandler, provideConfig))
}
