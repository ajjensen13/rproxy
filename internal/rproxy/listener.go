package rproxy

import (
	"golang.org/x/crypto/acme/autocert"
	"net"
)

func NewListener(cache autocert.Cache, domains []string) net.Listener {
	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      cache,
		HostPolicy: autocert.HostWhitelist(domains...),
	}
	return m.Listener()
}
