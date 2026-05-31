//go:build unix

package ipc

import (
	"context"
	"net"
	"net/http"
	"os"
)

var (
	unixMkdirAll  = os.MkdirAll
	unixRemove    = os.Remove
	unixNetListen = func(network, address string) (net.Listener, error) {
		var lc net.ListenConfig
		return lc.Listen(context.Background(), network, address)
	}
	unixChmod      = os.Chmod
	newHTTPRequest = http.NewRequestWithContext
	unixServeLoop  = func(srv *http.Server, ln net.Listener) error { return srv.Serve(ln) }
)
