//go:build linux

package ipc

import (
	"net"

	"golang.org/x/sys/unix"
)

func peerFromConn(conn net.Conn) (PeerContext, error) {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return PeerContext{}, ErrUnsupportedConn
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return PeerContext{}, err
	}
	var cred *unix.Ucred
	var ctlErr error
	if err := raw.Control(func(fd uintptr) {
		cred, ctlErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return PeerContext{}, err
	}
	if ctlErr != nil {
		return PeerContext{}, ctlErr
	}
	return PeerContext{
		Level: PeerLevelLocal,
		UID:   int(cred.Uid),
		GID:   int(cred.Gid),
		PID:   int(cred.Pid),
	}, nil
}
