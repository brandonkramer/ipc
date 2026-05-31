// Package ipc provides cross-platform local transport for service daemons.
//
// Use [Addr] to describe a local endpoint once, then [Listen], [Dial], and [DialTimeout]
// work on both Unix domain sockets and Windows named pipes.
//
// Low-level helpers ([ListenUnix], [ListenPipe], [PipeName]) are available when you
// manage addresses yourself. Optional HTTP-over-Unix helpers include [RunUnixHTTP] and
// [UnixHTTPClient].
//
// Example:
//
//	addr := ipc.Addr{
//	    Unix:       "/var/run/mysvc/rpc.sock",
//	    PipePrefix: "mysvc",
//	    PipeKey:    "/data/mysvc",
//	}
//	ln, err := ipc.Listen(addr)
package ipc
