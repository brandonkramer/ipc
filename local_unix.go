//go:build unix

package ipc

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Listen opens a listener for addr on this platform.
func Listen(addr Addr) (net.Listener, error) {
	if addr.Unix == "" {
		return nil, ErrUnixPathRequired
	}
	return ListenUnix(addr.Unix, false)
}

// Dial connects to addr on this platform.
func Dial(ctx context.Context, addr Addr) (net.Conn, error) {
	if addr.Unix == "" {
		return nil, ErrUnixPathRequired
	}
	return DialUnix(ctx, addr.Unix)
}

// DialTimeout connects to addr with a timeout bound to ctx.
func DialTimeout(ctx context.Context, addr Addr, timeout time.Duration) (net.Conn, error) {
	if addr.Unix == "" {
		return nil, ErrUnixPathRequired
	}
	return DialUnixTimeout(ctx, addr.Unix, timeout)
}

// ListenUnix opens a Unix domain socket listener at addr.
func ListenUnix(addr string, mkdirParent bool) (net.Listener, error) {
	if addr == "" {
		return nil, ErrUnixPathRequired
	}
	if mkdirParent {
		if err := os.MkdirAll(filepath.Dir(addr), 0o755); err != nil {
			return nil, fmt.Errorf("ipc: mkdir parent for %s: %w", addr, err)
		}
	}
	if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("ipc: remove stale socket %s: %w", addr, err)
	}
	ln, err := net.Listen("unix", addr) //nolint:noctx // local unix socket bind
	if err != nil {
		return nil, fmt.Errorf("ipc: listen unix %s: %w", addr, err)
	}
	if err := os.Chmod(addr, 0o600); err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("ipc: chmod unix %s: %w", addr, err)
	}
	return ln, nil
}

// DialUnix connects to a Unix domain socket at addr.
func DialUnix(ctx context.Context, addr string) (net.Conn, error) {
	if addr == "" {
		return nil, ErrUnixPathRequired
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", addr)
	if err != nil {
		return nil, fmt.Errorf("ipc: dial unix %s: %w", addr, err)
	}
	return conn, nil
}

// DialUnixTimeout connects to addr with a timeout bound to ctx.
func DialUnixTimeout(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
	if timeout <= 0 {
		return DialUnix(ctx, addr)
	}
	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return DialUnix(dctx, addr)
}
