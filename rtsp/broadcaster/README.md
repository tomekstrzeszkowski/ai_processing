# Provider

Send frames via p2p. Frames are pulled from watcher.

```
go build -o ./bin/provider ./cmd/provider/main.go
```
or
```
go run ./cmd/provider/main.go
```

# Viewer

Receive frames via p2p and display them in webpage served by simple server.

```
go build -o ./bin/viewer ./cmd/viewer/main.go
```

## WebRTC

### Signaling server

```
go build -o ./bin/web_rtc/signaling ./cmd/web_rtc_server/signaling.go
```

### Offeror

```
go build -o ./bin/web_rtc/offeror ./cmd/web_rtc_offeror/offeror.go
```


## Video creator

Watch shared memory, convert frames to video, manage memory.

```
go build -o ./bin/video_creator ./cmd/videoCreator/main.go
```
## Server

Serve frames by swapping image source.

```
go build -o ./bin/server ./cmd/server/main.go
```

# Testing

```
go test ./...
```