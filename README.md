# lightcable

lightweight websocket channel server

[![Go Reference](https://pkg.go.dev/badge/github.com/a-wing/lightcable.svg)](https://pkg.go.dev/github.com/a-wing/lightcable)
[![Build Status](https://github.com/a-wing/lightcable/workflows/ci/badge.svg)](https://github.com/a-wing/lightcable/actions?query=workflow%3Aci)
[![codecov](https://codecov.io/gh/a-wing/lightcable/branch/master/graph/badge.svg)](https://codecov.io/gh/a-wing/lightcable)
[![Go Report Card](https://goreportcard.com/badge/github.com/a-wing/lightcable)](https://goreportcard.com/report/github.com/a-wing/lightcable)
[![license](https://img.shields.io/github/license/a-wing/lightcable.svg?maxAge=2592000)](https://github.com/a-wing/lightcable/blob/master/LICENSE)
[![GitHub release](https://img.shields.io/github/tag/a-wing/lightcable.svg?label=release)](https://github.com/a-wing/lightcable/releases)

## Compile

* golang >= 1.13

### Application scenario

* A simple Websocket broadcast room
* WebRTC signaling server
* Replace [socket.io](https://socket.io/)  (Special case: client only join a room)
* Wx APP sdk and webview communication (WeChat Mini Program and Webview communication)

### As a Application

```bash
docker run -p 8080:8080 ghcr.io/a-wing/lightcable
```

install && run

```bash
go install github.com/a-wing/lightcable/cmd/lightcable

# Listen port: 8088
lightcable -l localhost:8088
```

### URL

```bash
ws://localhost:8080/{room}
```

### broadcast server demo

Server

```bash
lightcable
```

Room: `xxx`

```bash
websocat --linemode-strip-newlines 'ws://localhost:8080/xxx'
```

Room: `xxx`

```bash
websocat --linemode-strip-newlines 'ws://localhost:8080/xxx'
```

Room: `xxx-2`

```bash
websocat --linemode-strip-newlines 'ws://localhost:8080/xxx-2'
```

