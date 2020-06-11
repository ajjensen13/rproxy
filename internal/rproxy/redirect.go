package rproxy

import (
	"fmt"
	"github.com/ajjensen13/gke"
	"github.com/ajjensen13/urlutil"
	"net/http"
)

type redirectHandler struct {
	lg *gke.Logger
	urlutil.Rewriter
}

func (h *redirectHandler) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	src := r.URL
	srcStr := src.String()
	des, err := h.Rewrite(src)
	if err != nil {
		panic(h.lg.ErrorErr(fmt.Errorf("rproxy: error redirecting from %s by rule %s: %w", srcStr, string(h.Rewriter), err)))
	}

	desStr := des.String()
	h.lg.Infof("redirecting from %s to %s", srcStr, desStr)
	http.Redirect(wr, r, desStr, http.StatusTemporaryRedirect)
}

func NewRedirectHandler(lg *gke.Logger, rw urlutil.Rewriter) http.Handler {
	return &redirectHandler{lg: lg, Rewriter: rw}
}
