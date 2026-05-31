//go:build unix

package ipc

import (
	"net"
	"net/http"
	"os"
)

var (
	unixMkdirAll   = os.MkdirAll
	unixRemove     = os.Remove
	unixNetListen  = net.Listen
	unixChmod      = os.Chmod
	newHTTPRequest = http.NewRequestWithContext
	unixServeLoop  = func(srv *http.Server, ln net.Listener) error { return srv.Serve(ln) }
)
