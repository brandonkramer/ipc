//go:build darwin

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
	var cred *unix.Xucred
	var pid int
	var ctlErr error
	if err := raw.Control(func(fd uintptr) {
		cred, ctlErr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if ctlErr != nil {
			return
		}
		pid, ctlErr = unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID)
	}); err != nil {
		return PeerContext{}, err
	}
	if ctlErr != nil {
		return PeerContext{}, ctlErr
	}
	gid := 0
	if cred.Ngroups > 0 {
		gid = int(cred.Groups[0])
	}
	return PeerContext{
		Level: PeerLevelLocal,
		UID:   int(cred.Uid),
		GID:   gid,
		PID:   pid,
	}, nil
}
