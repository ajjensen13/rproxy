package rproxy

import (
	"fmt"
	"github.com/ajjensen13/urlutil"
	"log"
	"net/http"
)

type redirectHandler urlutil.Rewriter

func (h redirectHandler) ServeHTTP(wr http.ResponseWriter, r *http.Request) {
	src := r.URL
	srcStr := src.String()
	des, err := urlutil.Rewriter(h).Rewrite(src)
	if err != nil {
		log.Panic(fmt.Errorf("rproxy: error redirecting from %s by rule %s: %w", srcStr, string(urlutil.Rewriter(h)), err))
		return
	}

	desStr := des.String()
	log.Printf("rproxy: redirecting from %s to %s", srcStr, desStr)
	http.Redirect(wr, r, desStr, http.StatusTemporaryRedirect)
}

func NewRedirectHandler(rw urlutil.Rewriter) http.Handler {
	return redirectHandler(rw)
}
