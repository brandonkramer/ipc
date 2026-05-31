package ipc

import "errors"

//
// ────────────────────────────────────────
// sentinel errors.
//

// ErrUnixPathRequired is returned when a Unix socket path is missing.
var ErrUnixPathRequired = errors.New("ipc: unix socket path required")

// ErrPipeKeyRequired is returned when a Windows pipe key is missing.
var ErrPipeKeyRequired = errors.New("ipc: pipe key required")
