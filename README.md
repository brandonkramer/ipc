# ipc

Cross-platform local transport for service daemons: Unix domain sockets, Windows named pipes, detached child spawn, and optional HTTP-over-Unix helpers.

## Quick start

```go
addr := ipc.Addr{
    Unix:       "/var/run/mysvc/rpc.sock",
    PipePrefix: "mysvc",
    PipeKey:    "/data/mysvc-home",
}

ln, err := ipc.Listen(addr)
conn, err := ipc.Dial(ctx, addr)
ipc.SetDetach(cmd)
```

## With svcroot

```go
layout := svcroot.Layout{SocketName: "rpc.sock", PipePrefix: "mysvc"}
addr := ipc.Addr{
    Unix:       svcroot.Socket(root, &layout),
    PipePrefix: layout.WithDefaults().PipePrefix,
    PipeKey:    root,
}
```

## HTTP over Unix (Unix only)

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
go func() { _ = ipc.RunUnixHTTP(ctx, "/var/run/mysvc/observe.sock", true, mux) }()

client := ipc.NewUnixHTTPClient("/var/run/mysvc/observe.sock")
err := client.Get(ctx, "/status", &out)
```
