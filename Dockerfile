# syntax=docker/dockerfile:1
#
# Multi-stage build:
#   1. web    — vite/tailwind build of the React SPA
#   2. server — go build with the SPA copied into internal/webfs/dist so
#               webfs's go:embed picks it up at compile time
#   3. runtime — minimal alpine with the static binary
#
# Build:    docker build -t flowgent .
# Run:      docker run -p 8080:8080 -e DATABASE_URL=... -e FLOWGENT_CRED_KEY=... flowgent
#
# Local dev uses docker-compose.yml which wires Postgres + env automatically.

FROM node:20-alpine AS web
WORKDIR /web
# Install deps first so subsequent source-only edits hit the layer cache.
COPY web/package.json web/package-lock.json* ./
RUN npm ci --no-audit --no-fund || npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS server
WORKDIR /src
# Module deps before source so dependency changes invalidate the cache
# narrowly.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Drop the built SPA into the embed directory before compiling Go so
# webfs's //go:embed all:dist sees real assets, not the placeholder.
RUN rm -rf internal/webfs/dist
COPY --from=web /web/dist internal/webfs/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/flowgent ./cmd/flowgent

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S flowgent && adduser -S -G flowgent flowgent
COPY --from=server /out/flowgent /usr/local/bin/flowgent
EXPOSE 8080
USER flowgent
ENTRYPOINT ["/usr/local/bin/flowgent"]
