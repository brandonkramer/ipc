package ipc

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
)

const defaultPipePrefix = "ipc"

// Addr names a local RPC endpoint for the current platform.
// On Unix, Unix is the domain socket path. On Windows, PipePrefix and PipeKey
// derive a stable named pipe via [PipeName].
type Addr struct {
	// Unix is the filesystem path for a Unix domain socket.
	Unix string
	// PipePrefix is the Windows named-pipe namespace prefix.
	PipePrefix string
	// PipeKey is a stable identity hashed into the Windows pipe name.
	PipeKey string
}

// PipeName returns a stable Windows named-pipe path for prefix and key.
func PipeName(prefix, key string) string {
	if prefix == "" {
		prefix = defaultPipePrefix
	}
	sum := sha256.Sum256([]byte(filepath.Clean(key)))
	return `\\.\pipe\` + prefix + `-` + hex.EncodeToString(sum[:8])
}
