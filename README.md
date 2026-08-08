# Lion 

**English** · [简体中文](./README_zh-CN.md)

## Introduction

This project uses Golang and Vue, handling RDP and VNC connections. It is mainly based on [Apache Guacamole](http://guacamole.apache.org/)

## Configuration

Refer to the configuration file [config_example](config_example.yml)

## Build the image

```shell
docker build -t ghcr.io/matheus-marques-ft/lion .
```

## Docker start

```shell
docker run -d --name jms_lion -p 8081:8081 \
-v $(pwd)/data:/opt/lion/data \
-v $(pwd)/config.yml:/opt/lion/config.yml \
ghcr.io/matheus-marques-ft/lion
```

## Repository Layout

This repo builds the `lion` image consumed by [js-installer](https://github.com/matheus-marques-ft/js-installer)'s `compose/lion.yml` — it's JumpServer's Guacamole-protocol gateway, handling RDP/VNC connections, written in Go with a Vue 3 frontend.

- **`main.go` + `pkg/`** — the Go backend: `pkg/guacd` (Guacamole protocol client/tunnel), `pkg/session` (session recording/permissions/parsing), `pkg/tunnel` (WebSocket tunnel server, replay upload, clipboard policy), `pkg/gateway` (asset domain lookups), `pkg/middleware` (cookie/session auth), `pkg/config` (env-driven config, see `config_example.yml`).
- **`ui/`** — the Vue 3 + TypeScript frontend embedded into the Go binary at build time (`ui/dist` is copied into the final image); renders the RDP/VNC canvas, on-screen keyboard, clipboard, file manager, and session-sharing UI.
- **`Dockerfile-base`** / **`Dockerfile`** — two-stage build: `Dockerfile-base` installs Go + Node toolchains and downloads dependencies (published as the `lion-base` image, rebuilt only when `go.mod`/`ui/package.json`/`ui/yarn.lock` change); `Dockerfile` builds both the Go binary and the Vue UI, then assembles the final image on top of `jumpserver/guacd` (the upstream Guacamole daemon image — not part of this fork, consumed as-is) via `s6-overlay`.
- **`Dockerfile.guacd`** — a standalone Dockerfile for building just the `guacd` daemon image (used by `docker-compose.yaml.example` for local/standalone testing).
- **`s6-overlay/`** — process supervision scripts for running `guacd` and `lion` together in one container.

### CI → GHCR mapping

| Workflow | Publishes |
|---|---|
| `build-base-image.yml` | `ghcr.io/matheus-marques-ft/lion-base:<timestamp>` — triggered by `go.mod`/`ui/package.json`/`ui/yarn.lock`/`Dockerfile-base` changes on `pr*` branches, then auto-commits the new tag into `Dockerfile` |
| `build-ghcr-image.yml` | `ghcr.io/matheus-marques-ft/lion:<tag>` — triggered on `v*` tags |
| `release-drafter.yml` | drafts a GitHub Release with the `make all` build artifacts — triggered on `v*` tags |
