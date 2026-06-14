# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/flowgent ./cmd/flowgent

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/flowgent /usr/local/bin/flowgent
EXPOSE 8080
USER nobody
ENTRYPOINT ["/usr/local/bin/flowgent"]
