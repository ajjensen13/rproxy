package rproxy

import (
	"golang.org/x/crypto/acme/autocert"
	"os"
	"path/filepath"
)

func NewDirCache() autocert.Cache {
	dir := filepath.Join(os.TempDir(), "golang-autocert")
	return autocert.DirCache(dir)
}
