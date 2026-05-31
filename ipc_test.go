package ipc_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/brandonkramer/ipc"
)

func testAddr(t *testing.T) (addr ipc.Addr, observe string) {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "ipc-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	sessions := filepath.Join(root, "sessions")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatal(err)
	}
	return ipc.Addr{
		Unix:       filepath.Join(sessions, "rpc.sock"),
		PipePrefix: "testsvc",
		PipeKey:    root,
	}, filepath.Join(sessions, "observe.sock")
}

func startUnixHTTPServer(t *testing.T, addr string, handler http.Handler) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = ipc.RunUnixHTTP(ctx, addr, true, handler)
	}()
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})

	deadline := time.Now().Add(2 * time.Second)
	client := ipc.NewUnixHTTPClient(addr)
	for time.Now().Before(deadline) {
		var out map[string]string
		if err := client.Get(context.Background(), "/ready", &out); err == nil {
			return cancel
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("unix http listener not ready")
	return cancel
}

func TestListenDial(t *testing.T) {
	t.Parallel()

	addr, _ := testAddr(t)
	ln, err := ipc.Listen(addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	ctx := context.Background()
	conn, err := ipc.Dial(ctx, addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	conn2, err := ipc.DialTimeout(ctx, addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn2.Close() })
}

func TestAddrValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		addr    ipc.Addr
		wantErr error
	}{
		{
			name:    "listen missing unix path",
			addr:    ipc.Addr{},
			wantErr: ipc.ErrUnixPathRequired,
		},
		{
			name:    "dial missing unix path",
			addr:    ipc.Addr{},
			wantErr: ipc.ErrUnixPathRequired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			switch tc.name {
			case "listen missing unix path":
				_, err := ipc.Listen(tc.addr)
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Listen() err=%v want %v", err, tc.wantErr)
				}
			case "dial missing unix path":
				_, err := ipc.Dial(context.Background(), tc.addr)
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Dial() err=%v want %v", err, tc.wantErr)
				}
			}
		})
	}
}

func TestPipeNameStable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		prefix string
		key    string
	}{
		{name: "explicit prefix", prefix: "svc", key: "/tmp/home"},
		{name: "default prefix", prefix: "", key: "/tmp/home"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := ipc.PipeName(tc.prefix, tc.key)
			b := ipc.PipeName(tc.prefix, tc.key)
			if a != b {
				t.Fatalf("unstable pipe: %q vs %q", a, b)
			}
			if a == "" {
				t.Fatal("expected non-empty pipe name")
			}
		})
	}
}

func TestUnixHTTPClientAndHelpers(t *testing.T) {
	_, observe := testAddr(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		ipc.HandleJSON(w, r, func() any { return map[string]string{"ok": "1"} })
	})
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		ipc.HandleJSON(w, r, func() any { return map[string]string{"x": "1"} })
	})
	mux.HandleFunc("/err", func(w http.ResponseWriter, r *http.Request) {
		ipc.HandleJSONErr(w, r, func() (any, error) { return nil, errBoom{} })
	})
	mux.HandleFunc("/prefix/", func(w http.ResponseWriter, r *http.Request) {
		ipc.HandlePrefix(w, r, "/prefix/", func(id string) (any, error) {
			if id == "missing" {
				return nil, errBoom{}
			}
			return map[string]string{"id": id}, nil
		})
	})
	mux.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		ipc.GETOnly(w, r, func(w http.ResponseWriter, _ *http.Request) {
			ipc.WriteJSON(w, map[string]int{"n": 1})
		})
	})

	startUnixHTTPServer(t, observe, mux)
	c := ipc.NewUnixHTTPClient(observe)
	var out map[string]string

	if err := c.Get(context.Background(), "/ok", &out); err != nil || out["x"] != "1" {
		t.Fatalf("/ok: out=%+v err=%v", out, err)
	}
	c.Timeout = 0
	if err := c.Get(context.Background(), "/ok", &out); err != nil {
		t.Fatal(err)
	}
	if err := c.Get(context.Background(), "/nope", &out); err == nil {
		t.Fatal("expected 404")
	}
	if err := c.Get(context.Background(), "/err", &out); err == nil {
		t.Fatal("expected 500")
	}
	if err := c.Get(context.Background(), "/prefix/id1", &out); err != nil || out["id"] != "id1" {
		t.Fatalf("prefix: %+v err=%v", out, err)
	}
	if err := c.Get(context.Background(), "/prefix/missing", &out); err == nil {
		t.Fatal("expected prefix error")
	}
	if err := c.Get(context.Background(), "/prefix/", &out); err == nil {
		t.Fatal("expected empty id 404")
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/post", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestRunUnixHTTPStopsOnCancel(t *testing.T) {
	_, observe := testAddr(t)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = ipc.RunUnixHTTP(ctx, observe, true, http.NewServeMux())
	}()

	time.Sleep(50 * time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after cancel")
	}
}

func TestSetDetach(t *testing.T) {
	t.Parallel()

	cmd := exec.CommandContext(context.Background(), "sleep", "0")
	ipc.SetDetach(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("missing SysProcAttr")
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }

func TestWriteJSONDirect(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	ipc.WriteJSON(rec, map[string]int{"a": 1})
	var out map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out["a"] != 1 {
		t.Fatalf("body=%s err=%v", rec.Body.String(), err)
	}
}

func TestDialTimeoutZero(t *testing.T) {
	t.Parallel()

	addr, _ := testAddr(t)
	ln, err := ipc.Listen(addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	conn, err := ipc.DialTimeout(context.Background(), addr, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
}

func TestListenUnixExistingParent(t *testing.T) {
	t.Parallel()

	_, observe := testAddr(t)
	ln1, err := ipc.ListenUnix(observe, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := ln1.Close(); err != nil {
		t.Fatal(err)
	}

	ln2, err := ipc.ListenUnix(observe, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln2.Close() })
}

func TestUnixHTTPClientDialError(t *testing.T) {
	t.Parallel()

	_, observe := testAddr(t)
	c := ipc.NewUnixHTTPClient(observe)
	c.Timeout = time.Millisecond
	var v map[string]string
	if err := c.Get(context.Background(), "/nope", &v); err == nil {
		t.Fatal("expected dial error")
	}
}

func TestListenUnixEmptyPath(t *testing.T) {
	t.Parallel()

	_, err := ipc.ListenUnix("", false)
	if !errors.Is(err, ipc.ErrUnixPathRequired) {
		t.Fatalf("err=%v", err)
	}
}

func TestUnixHTTPClientEmptyAddr(t *testing.T) {
	t.Parallel()

	c := ipc.NewUnixHTTPClient("")
	var v map[string]string
	if err := c.Get(context.Background(), "/nope", &v); !errors.Is(err, ipc.ErrUnixPathRequired) {
		t.Fatalf("err=%v", err)
	}
}

func TestDialUnixEmptyPath(t *testing.T) {
	t.Parallel()

	_, err := ipc.DialUnix(context.Background(), "")
	if !errors.Is(err, ipc.ErrUnixPathRequired) {
		t.Fatalf("err=%v", err)
	}
}
