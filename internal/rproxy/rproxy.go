package rproxy

import (
	"fmt"
	"github.com/ajjensen13/urlutil"
	"log"
	"net/http"
	"net/http/httputil"
)

func NewReverseProxyHandler(rw urlutil.Rewriter) http.Handler {
	return &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			src := r.URL
			srcStr := src.String()
			des, err := urlutil.Rewriter(rw).Rewrite(src)
			if err != nil {
				log.Panic(fmt.Errorf("rproxy: error proxying from %s by rule %s: %w", srcStr, string(urlutil.Rewriter(rw)), err))
				return
			}

			log.Printf("rproxy: proxying from %s to %s", srcStr, des.String())
			r.URL = des
		},
	}
}
