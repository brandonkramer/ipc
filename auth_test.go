package ipc_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/brandonkramer/ipc"
)

func TestTrustedPeer(t *testing.T) {
	t.Parallel()

	c := ipc.TrustedPeer()
	if !c.CanWrite() || c.UID != os.Getuid() {
		t.Fatalf("ctx=%+v", c)
	}
}

func TestPeerFromUnixConn(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "ipc-auth-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	addr := filepath.Join(root, "peer.sock")
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "unix", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	clientReady := make(chan net.Conn, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		clientReady <- conn
	}()

	dialer := net.Dialer{}
	client, err := dialer.DialContext(context.Background(), "unix", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	server := <-clientReady
	t.Cleanup(func() { _ = server.Close() })

	got, err := ipc.PeerFromConn(server)
	if err != nil {
		t.Fatal(err)
	}
	if !got.CanWrite() || got.UID != os.Getuid() {
		t.Fatalf("ctx=%+v", got)
	}
}

func TestPeerUnsupportedConn(t *testing.T) {
	t.Parallel()

	c1, c2 := net.Pipe()
	t.Cleanup(func() { _ = c1.Close(); _ = c2.Close() })
	if _, err := ipc.PeerFromConn(c1); err == nil {
		t.Fatal("expected unsupported connection error")
	}
}
