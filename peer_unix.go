//go:build unix && !linux && !darwin

package ipc

import "net"

func peerFromConn(conn net.Conn) (PeerContext, error) {
	if _, ok := conn.(*net.UnixConn); !ok {
		return PeerContext{}, ErrUnsupportedConn
	}
	return PeerContext{Level: PeerLevelLocal}, nil
}
