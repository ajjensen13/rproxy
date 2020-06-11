package rproxy

import (
	"fmt"
	"github.com/ajjensen13/gke"
	"github.com/ajjensen13/urlutil"
	"net/http"
	"net/http/httputil"
	"time"
)

func NewReverseProxyHandler(lg *gke.Logger, rw urlutil.Rewriter) http.Handler {
	return &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			src := r.URL
			srcStr := src.String()
			des, err := urlutil.Rewriter(rw).Rewrite(src)
			if err != nil {
				panic(lg.ErrorErr(fmt.Errorf("error proxying from %s by rule %s: %w", srcStr, string(urlutil.Rewriter(rw)), err)))
			}

			lg.Infof("proxying from %s to %s", srcStr, des.String())
			r.URL = des
		},
		FlushInterval: time.Second,
	}
}
