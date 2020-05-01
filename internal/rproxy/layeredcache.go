package rproxy

import (
	"context"
	"fmt"
	"golang.org/x/crypto/acme/autocert"
	"log"
	"time"
)

func NewLayeredCache(layers ...autocert.Cache) autocert.Cache {
	return &layeredCache{layers: layers}
}

type layeredCache struct {
	layers []autocert.Cache
}

func (l *layeredCache) Get(ctx context.Context, key string) (retval []byte, reterr error) {
	misses := 0
	start := time.Now()
	defer func() {
		duration := time.Now().Sub(start)
		log.Printf("rproxy: getting key %q completed in %v [cache misses: %d, error: %v]", key, duration, misses, reterr)
	}()

	defer func() {
		if reterr != nil {
			log.Print(fmt.Errorf("rproxy: error encountered while getting key %q. skipping backtracking update: %w", key, reterr))
			return
		}
		if misses < 1 {
			return
		}

		log.Printf("rproxy: performing backtracking update at key %q of %d layers", key, misses)
		reterr = doPut(ctx, l.layers[:misses], key, retval)
		if reterr != nil {
			retval = nil
			reterr = fmt.Errorf("rproxy: error while backtracking update: %w", reterr)
		}
	}()

	for layer, cache := range l.layers {
		retval, reterr = cache.Get(ctx, key)
		if reterr == autocert.ErrCacheMiss {
			misses++
			continue
		}
		if reterr != nil {
			reterr = fmt.Errorf("rproxy: error getting value at key %q in layer %d (%T): %w", key, layer, cache, reterr)
			return
		}

		log.Printf("rproxy: got %d bytes at key %q in layer %d (%T)", len(retval), key, layer, cache)
		return
	}

	return
}

func (l *layeredCache) Put(ctx context.Context, key string, val []byte) (err error) {
	start := time.Now()
	defer func() {
		duration := time.Now().Sub(start)
		log.Printf("rproxy: putting %d bytes at key %q completed in %v [error: %v]", len(val), key, duration, err)
	}()
	return doPut(ctx, l.layers, key, val)
}

func doPut(ctx context.Context, layers []autocert.Cache, key string, data []byte) error {
	for layer, cache := range layers {
		err := cache.Put(ctx, key, data)
		if err != nil {
			return fmt.Errorf("rproxy: error updating cert cache layer %d (%T) for key %s: %v", layer, cache, key, err)
		}
		log.Printf("rproxy: put %d bytes at key %q in layer %d (%T)", len(data), key, layer, cache)
	}
	return nil
}

func (l *layeredCache) Delete(ctx context.Context, key string) (err error) {
	start := time.Now()
	defer func() {
		duration := time.Now().Sub(start)
		log.Printf("rproxy: deleting key %q completed in %v [error: %v]", key, duration, err)
	}()

	for layer, cache := range l.layers {
		err := cache.Delete(ctx, key)
		if err != nil {
			return fmt.Errorf("rproxy: error deleting cert cache layer %d for key %s: %v", layer, key, err)
		}
		log.Printf("rproxy: deleted key %q in layer %d (%T)", key, layer, cache)
	}
	return nil
}
