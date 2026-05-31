package ipc

import (
	"errors"
	"net"
	"os"
)

//
// ────────────────────────────────────────
// local peer authorization.
//

// PeerLevel describes how strongly a caller is trusted.
type PeerLevel int

const (
	// PeerLevelUnknown means peer credentials could not be established.
	PeerLevelUnknown PeerLevel = iota
	// PeerLevelLocal means the peer belongs to the current OS user.
	PeerLevelLocal
	// PeerLevelTrusted marks in-process or explicitly trusted callers.
	PeerLevelTrusted
)

// ErrAccessDenied is returned when a caller lacks permission.
var ErrAccessDenied = errors.New("ipc: access denied")

// ErrUnsupportedConn is returned for connection types without peer credentials.
var ErrUnsupportedConn = errors.New("ipc: unsupported connection type")

// PeerContext carries caller identity for RPC authorization.
type PeerContext struct {
	Level PeerLevel
	UID   int
	GID   int
	PID   int
}

// TrustedPeer returns a context for in-process trusted callers.
func TrustedPeer() PeerContext {
	return PeerContext{
		Level: PeerLevelTrusted,
		UID:   os.Getuid(),
		GID:   os.Getgid(),
		PID:   os.Getpid(),
	}
}

// IsLocal reports whether the caller is trusted or matches the current uid.
func (c PeerContext) IsLocal() bool {
	if c.Level == PeerLevelTrusted {
		return true
	}
	return c.Level == PeerLevelLocal && c.UID == os.Getuid()
}

// CanRead reports whether read-only RPC is allowed.
func (c PeerContext) CanRead() bool {
	return c.IsLocal()
}

// CanWrite reports whether mutating RPC is allowed.
func (c PeerContext) CanWrite() bool {
	return c.IsLocal()
}

// PeerFromConn derives caller context from a local transport connection.
func PeerFromConn(conn net.Conn) (PeerContext, error) {
	return peerFromConn(conn)
}
