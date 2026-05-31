//go:build darwin

package ipc

import (
	"net"
	"unsafe"

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
	var cred unix.Ucred
	var pid int
	var ctlErr error
	if err := raw.Control(func(fd uintptr) {
		n := uint32(unsafe.Sizeof(cred))
		_, _, e := unix.Syscall6(unix.SYS_GETSOCKOPT, fd, unix.SOL_LOCAL, unix.LOCAL_PEERCRED,
			uintptr(unsafe.Pointer(&cred)), uintptr(unsafe.Pointer(&n)), 0)
		if e != 0 {
			ctlErr = e
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
