//go:build windows

package ipc

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var errUnixHTTPUnavailable = errors.New("ipc: unix domain HTTP is not available on Windows")

// RunUnixHTTP listens on addr and serves handler until ctx is cancelled or Serve fails.
func RunUnixHTTP(ctx context.Context, addr string, mkdirParent bool, handler http.Handler) error {
	_ = ctx
	_ = addr
	_ = mkdirParent
	_ = handler
	return errUnixHTTPUnavailable
}

// UnixHTTPClient performs read-only GET requests over a Unix domain socket.
type UnixHTTPClient struct {
	Addr    string
	Timeout time.Duration
}

// NewUnixHTTPClient returns a client for addr.
func NewUnixHTTPClient(addr string) *UnixHTTPClient {
	return &UnixHTTPClient{Addr: addr}
}

// Get decodes a GET response body into dest.
func (c *UnixHTTPClient) Get(ctx context.Context, path string, dest any) error {
	_ = ctx
	_ = path
	_ = dest
	return errUnixHTTPUnavailable
}
