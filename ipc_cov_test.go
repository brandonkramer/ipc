//go:build unix

package ipc

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type unixStub struct {
	mkdirErr  error
	removeErr error
	listenErr error
	chmodErr  error
}

func shortUnixAddr(t *testing.T, name string) string {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "ipc-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return filepath.Join(root, name)
}

func stubUnixHooks(t *testing.T, s unixStub) {
	t.Helper()
	prevMkdir := unixMkdirAll
	prevRemove := unixRemove
	prevListen := unixNetListen
	prevChmod := unixChmod
	unixMkdirAll = func(path string, perm os.FileMode) error {
		if s.mkdirErr != nil {
			return s.mkdirErr
		}
		return prevMkdir(path, perm)
	}
	unixRemove = func(name string) error {
		if s.removeErr != nil {
			return s.removeErr
		}
		return prevRemove(name)
	}
	unixNetListen = func(network, address string) (net.Listener, error) {
		if s.listenErr != nil {
			return nil, s.listenErr
		}
		return prevListen(network, address)
	}
	unixChmod = func(name string, perm os.FileMode) error {
		if s.chmodErr != nil {
			return s.chmodErr
		}
		return prevChmod(name, perm)
	}
	t.Cleanup(func() {
		unixMkdirAll = prevMkdir
		unixRemove = prevRemove
		unixNetListen = prevListen
		unixChmod = prevChmod
	})
}

func TestListenUnixInjectedErrors(t *testing.T) {
	addr := shortUnixAddr(t, "rpc.sock")
	errMkdir := errors.New("mkdir failed")
	errRemove := errors.New("remove failed")
	errListen := errors.New("listen failed")
	errChmod := errors.New("chmod failed")

	cases := []struct {
		name string
		stub unixStub
	}{
		{name: "mkdir", stub: unixStub{mkdirErr: errMkdir}},
		{name: "remove", stub: unixStub{removeErr: errRemove}},
		{name: "listen", stub: unixStub{listenErr: errListen}},
		{name: "chmod", stub: unixStub{chmodErr: errChmod}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubUnixHooks(t, tc.stub)
			_, err := ListenUnix(addr, true)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestListenUnixEmptyAddr(t *testing.T) {
	_, err := ListenUnix("", false)
	if !errors.Is(err, ErrUnixPathRequired) {
		t.Fatalf("err=%v", err)
	}
}

func TestListenUnixSuccessNoMkdir(t *testing.T) {
	addr := shortUnixAddr(t, "rpc.sock")
	ln, err := ListenUnix(addr, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
}

func TestDialTimeoutEmptyAddr(t *testing.T) {
	t.Parallel()

	_, err := DialTimeout(context.Background(), Addr{}, time.Second)
	if !errors.Is(err, ErrUnixPathRequired) {
		t.Fatalf("err=%v", err)
	}
}

func TestDialUnixFailure(t *testing.T) {
	t.Parallel()

	_, err := DialUnix(context.Background(), filepath.Join(t.TempDir(), "missing.sock"))
	if err == nil {
		t.Fatal("expected dial error")
	}
}

func TestHandleJSONErrSuccess(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ok", http.NoBody)
	HandleJSONErr(rec, req, func() (any, error) {
		return map[string]int{"n": 1}, nil
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRunUnixHTTPListenError(t *testing.T) {
	err := RunUnixHTTP(context.Background(), "", true, http.NewServeMux())
	if !errors.Is(err, ErrUnixPathRequired) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunUnixHTTPErrOnServe(t *testing.T) {
	serveErr := errors.New("serve failed")
	prev := unixServeLoop
	unixServeLoop = func(*http.Server, net.Listener) error { return serveErr }
	t.Cleanup(func() { unixServeLoop = prev })

	addr := shortUnixAddr(t, "observe.sock")
	err := RunUnixHTTP(context.Background(), addr, true, http.NewServeMux())
	if err == nil || !strings.Contains(err.Error(), "serve failed") {
		t.Fatalf("err=%v", err)
	}
}

func TestRunUnixHTTPErrOnCancel(t *testing.T) {
	serveErr := errors.New("serve failed")
	ctx, cancel := context.WithCancel(context.Background())
	prev := unixServeLoop
	unixServeLoop = func(*http.Server, net.Listener) error {
		<-ctx.Done()
		return serveErr
	}
	t.Cleanup(func() { unixServeLoop = prev })

	addr := shortUnixAddr(t, "observe.sock")
	done := make(chan error, 1)
	go func() { done <- RunUnixHTTP(ctx, addr, true, http.NewServeMux()) }()
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "serve failed") {
			t.Fatalf("err=%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server stop")
	}
}

func TestRunUnixHTTPErrServerClosed(t *testing.T) {
	prev := unixServeLoop
	unixServeLoop = func(*http.Server, net.Listener) error { return http.ErrServerClosed }
	t.Cleanup(func() { unixServeLoop = prev })

	addr := shortUnixAddr(t, "observe.sock")
	err := RunUnixHTTP(context.Background(), addr, true, http.NewServeMux())
	if err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestUnixHTTPClientBadJSON(t *testing.T) {
	addr := shortUnixAddr(t, "observe.sock")
	mux := http.NewServeMux()
	mux.HandleFunc("/bad", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	})

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = RunUnixHTTP(ctx, addr, true, mux)
	}()
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	deadline := time.Now().Add(2 * time.Second)
	client := NewUnixHTTPClient(addr)
	for time.Now().Before(deadline) {
		var v map[string]string
		err := client.Get(context.Background(), "/bad", &v)
		if err != nil && strings.Contains(err.Error(), "decode GET") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected decode error")
}

func TestUnixHTTPClientNewRequestError(t *testing.T) {
	prev := newHTTPRequest
	newHTTPRequest = func(context.Context, string, string, io.Reader) (*http.Request, error) {
		return nil, errors.New("new request failed")
	}
	t.Cleanup(func() { newHTTPRequest = prev })

	c := NewUnixHTTPClient(shortUnixAddr(t, "s.sock"))
	var v map[string]string
	err := c.Get(context.Background(), "/ok", &v)
	if err == nil || !strings.Contains(err.Error(), "new GET") {
		t.Fatalf("err=%v", err)
	}
}
