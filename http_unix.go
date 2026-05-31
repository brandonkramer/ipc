//go:build unix

package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const defaultHTTPClientTimeout = 5 * time.Second

// RunUnixHTTP listens on addr and serves handler until ctx is cancelled or Serve fails.
func RunUnixHTTP(ctx context.Context, addr string, mkdirParent bool, handler http.Handler) error {
	ln, err := ListenUnix(addr, mkdirParent)
	if err != nil {
		return err
	}
	defer ln.Close()

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: defaultHTTPClientTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- unixServeLoop(srv, ln)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultHTTPClientTimeout)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("ipc: unix http serve %s: %w", addr, err)
		}
		return ctx.Err()
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("ipc: unix http serve %s: %w", addr, err)
		}
		return nil
	}
}

// UnixHTTPClient performs read-only GET requests over a Unix domain socket.
type UnixHTTPClient struct {
	// Addr is the Unix domain socket path.
	Addr string
	// Timeout bounds each request when non-zero.
	Timeout time.Duration
}

// NewUnixHTTPClient returns a client for addr.
func NewUnixHTTPClient(addr string) *UnixHTTPClient {
	return &UnixHTTPClient{Addr: addr, Timeout: defaultHTTPClientTimeout}
}

// Get decodes a GET response body into dest.
func (c *UnixHTTPClient) Get(ctx context.Context, path string, dest any) error {
	if c.Addr == "" {
		return ErrUnixPathRequired
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultHTTPClientTimeout
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{DialContext: func(dctx context.Context, _, _ string) (net.Conn, error) {
			return DialUnix(dctx, c.Addr)
		}},
	}
	req, err := newHTTPRequest(ctx, http.MethodGet, "http://local"+path, http.NoBody)
	if err != nil {
		return fmt.Errorf("ipc: new GET %s: %w", path, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ipc: GET %s via %s: %w", path, c.Addr, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ipc: GET %s: %s", path, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("ipc: decode GET %s: %w", path, err)
	}
	return nil
}
