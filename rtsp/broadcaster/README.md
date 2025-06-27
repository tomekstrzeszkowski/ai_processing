# Provider

Send frames via p2p. Frames are pulled from watcher.

```
go build -o ./bin/provider ./cmd/provider/main.go
```

# Viewer

Receive frames via p2p and display them in webpage served by simple server.

```
go build -o ./bin/viewer ./cmd/viewer/main.go
```

## WebRTC

**Still in progress**. Sending frames works via WS. WebRTC part still needs some work.

```
go build -o ./bin/webrtc ./cmd/webRTC/main.go
```

## Watcher

Watch shared memory.

```
go build -o ./bin/watcher ./cmd/watcher/main.go
```
## Server

Serve frames by swapping image source.

```
go build -o ./bin/server ./cmd/server/main.go
```