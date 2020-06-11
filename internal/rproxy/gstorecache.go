package rproxy

import (
	"bytes"
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"github.com/ajjensen13/gke"
	"golang.org/x/crypto/acme/autocert"
	"io"
	"io/ioutil"
	"path"
)

func NewGStorageCache(lg gke.Logger, bucketHandle *storage.BucketHandle) autocert.Cache {
	return &gStorageCache{lg: lg, bucketHandle: bucketHandle}
}

type gStorageCache struct {
	lg           gke.Logger
	bucketHandle *storage.BucketHandle
}

func (g *gStorageCache) Get(ctx context.Context, key string) ([]byte, error) {
	obj := g.bucketHandle.Object(path.Join("golang-autocert", key))
	g.lg.Infof("get %q from gstore %s/%s", key, obj.BucketName(), obj.ObjectName())

	reader, err := obj.NewReader(ctx)
	if err == storage.ErrObjectNotExist {
		return nil, autocert.ErrCacheMiss
	}

	if err != nil {
		return nil, fmt.Errorf("rproxy: error while getting %q from gstorage cache: %w", key, err)
	}
	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("rproxy: error while reading %q from gstorage cache: %w", key, err)
	}

	return data, nil
}

func (g *gStorageCache) Put(ctx context.Context, key string, data []byte) (reterr error) {
	obj := g.bucketHandle.Object(path.Join("golang-autocert", key))
	g.lg.Infof("put %d bytes at%q from gstore %s/%s", len(data), key, obj.BucketName(), obj.ObjectName())

	des := obj.NewWriter(ctx)
	defer func() {
		err := des.Close()
		if err != nil && reterr == nil {
			reterr = err
		}
	}()

	src := bytes.NewReader(data)

	_, reterr = io.Copy(des, src)
	if reterr != nil {
		return fmt.Errorf("rproxy: error while writing %q to gstorage cache: %w", key, reterr)
	}

	return nil
}

func (g *gStorageCache) Delete(ctx context.Context, key string) error {
	obj := g.bucketHandle.Object(key)
	return obj.Delete(ctx)
}
