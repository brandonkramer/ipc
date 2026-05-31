//go:build windows

package ipc

import (
	"context"
	"fmt"
	"net"
	"time"

	winio "github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

// Listen opens a listener for addr on this platform.
func Listen(addr Addr) (net.Listener, error) {
	if addr.PipeKey == "" {
		return nil, ErrPipeKeyRequired
	}
	return ListenPipe(PipeName(addr.PipePrefix, addr.PipeKey))
}

// Dial connects to addr on this platform.
func Dial(ctx context.Context, addr Addr) (net.Conn, error) {
	if addr.PipeKey == "" {
		return nil, ErrPipeKeyRequired
	}
	return DialPipe(ctx, PipeName(addr.PipePrefix, addr.PipeKey))
}

// DialTimeout connects to addr with a timeout bound to ctx.
func DialTimeout(ctx context.Context, addr Addr, timeout time.Duration) (net.Conn, error) {
	if addr.PipeKey == "" {
		return nil, ErrPipeKeyRequired
	}
	return DialPipeTimeout(ctx, PipeName(addr.PipePrefix, addr.PipeKey), timeout)
}

// ListenPipe opens a Windows named-pipe listener at pipe.
func ListenPipe(pipe string) (net.Listener, error) {
	cfg := &winio.PipeConfig{}
	if sd, err := pipeSecuritySDDL(); err == nil {
		cfg.SecurityDescriptor = sd
	}
	ln, err := winio.ListenPipe(pipe, cfg)
	if err != nil {
		return nil, fmt.Errorf("ipc: listen pipe %s: %w", pipe, err)
	}
	return ln, nil
}

// DialPipe connects to a Windows named pipe.
func DialPipe(ctx context.Context, pipe string) (net.Conn, error) {
	conn, err := winio.DialPipeContext(ctx, pipe)
	if err != nil {
		return nil, fmt.Errorf("ipc: dial pipe %s: %w", pipe, err)
	}
	return conn, nil
}

// DialPipeTimeout connects to pipe with a timeout bound to ctx.
func DialPipeTimeout(ctx context.Context, pipe string, timeout time.Duration) (net.Conn, error) {
	if timeout <= 0 {
		return DialPipe(ctx, pipe)
	}
	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return DialPipe(dctx, pipe)
}

func pipeSecuritySDDL() (string, error) {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return "", err
	}
	defer token.Close()

	user, err := token.GetTokenUser()
	if err != nil {
		return "", err
	}
	sid := user.User.Sid.String()
	return fmt.Sprintf("D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;%s)", sid), nil
}
