//go:build windows

package ipc

import "net"

func peerFromConn(conn net.Conn) (PeerContext, error) {
	if conn == nil {
		return PeerContext{}, ErrUnsupportedConn
	}
	return PeerContext{Level: PeerLevelLocal}, nil
}
