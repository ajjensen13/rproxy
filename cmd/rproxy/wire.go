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
	"context"
	"github.com/ajjensen13/gke"
	"github.com/google/wire"
	"net"
	"net/http"
)

func newLogger(ctx context.Context) (*gke.Logger, func()) {
	panic(wire.Build(wire.Value("rproxy"), provideLogger, provideLogClient))
}

func newListener(ctx context.Context, l *gke.Logger) (net.Listener, func()) {
	panic(wire.Build(provideListener, provideAutocertCache, provideBucketHandle, provideStorageClient, provideDomains, provideConfig))
}

func newServer(ctx context.Context, l *gke.Logger) *http.Server {
	panic(wire.Build(provideServer, provideErrorLogger, provideHandler, provideConfig))
}
