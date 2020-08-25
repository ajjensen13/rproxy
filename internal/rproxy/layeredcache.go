package rproxy

import (
	"context"
	"fmt"
	"github.com/ajjensen13/gke"
	"golang.org/x/crypto/acme/autocert"
	"time"
)

func NewLayeredCache(lg gke.Logger, layers ...autocert.Cache) autocert.Cache {
	return &layeredCache{lg: lg, layers: layers}
}

type layeredCache struct {
	lg     gke.Logger
	layers []autocert.Cache
}

func (c *layeredCache) Get(ctx context.Context, key string) (retval []byte, reterr error) {
	misses := 0
	start := time.Now()
	defer func() {
		duration := time.Now().Sub(start)
		c.lg.Defaultf("getting key %q completed in %v [cache misses: %d, error: %v]", key, duration, misses, reterr)
	}()

	defer func() {
		if reterr == autocert.ErrCacheMiss {
			c.lg.Defaultf("key %q not found in layered cache", key)
			return
		}
		if reterr != nil {
			reterr = c.lg.ErrorErr(fmt.Errorf("error encountered while getting key %q. skipping backtracking update: %w", key, reterr))
			return
		}
		if misses < 1 {
			return
		}

		c.lg.Defaultf("performing backtracking update at key %q of %d layers", key, misses)
		reterr = doPut(ctx, c.lg, c.layers[:misses], key, retval)
		if reterr != nil {
			retval = nil
			reterr = fmt.Errorf("rproxy: error while backtracking update: %w", reterr)
		}
	}()

	for layer, cache := range c.layers {
		retval, reterr = cache.Get(ctx, key)
		if reterr == autocert.ErrCacheMiss {
			misses++
			continue
		}
		if reterr != nil {
			reterr = fmt.Errorf("rproxy: error getting value at key %q in layer %d (%T): %w", key, layer, cache, reterr)
			return
		}

		c.lg.Defaultf("got %d bytes at key %q in layer %d (%T)", len(retval), key, layer, cache)
		return
	}

	return nil, autocert.ErrCacheMiss
}

func (c *layeredCache) Put(ctx context.Context, key string, val []byte) (err error) {
	start := time.Now()
	defer func() {
		duration := time.Now().Sub(start)
		c.lg.Defaultf("putting %d bytes at key %q completed in %v [error: %v]", len(val), key, duration, err)
	}()
	return doPut(ctx, c.lg, c.layers, key, val)
}

func doPut(ctx context.Context, lg gke.Logger, layers []autocert.Cache, key string, data []byte) error {
	for layer, cache := range layers {
		err := cache.Put(ctx, key, data)
		if err != nil {
			return fmt.Errorf("rproxy: error updating cert cache layer %d (%T) for key %s: %v", layer, cache, key, err)
		}
		lg.Defaultf("put %d bytes at key %q in layer %d (%T)", len(data), key, layer, cache)
	}
	return nil
}

func (c *layeredCache) Delete(ctx context.Context, key string) (err error) {
	start := time.Now()
	defer func() {
		duration := time.Now().Sub(start)
		c.lg.Defaultf("deleting key %q completed in %v [error: %v]", key, duration, err)
	}()

	for layer, cache := range c.layers {
		err := cache.Delete(ctx, key)
		if err != nil {
			return fmt.Errorf("rproxy: error deleting cert cache layer %d for key %s: %v", layer, key, err)
		}
		c.lg.Defaultf("deleted key %q in layer %d (%T)", key, layer, cache)
	}
	return nil
}
