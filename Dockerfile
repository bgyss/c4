# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN set -eux; \
	GOARM=""; \
	if [ "${TARGETARCH}" = "arm" ] && [ -n "${TARGETVARIANT}" ]; then GOARM="${TARGETVARIANT#v}"; fi; \
	CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" GOARM="${GOARM}" \
		go build -trimpath -ldflags='-s -w' -o /out/c4 ./cmd/c4

FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /data

COPY --from=builder /out/c4 /usr/local/bin/c4

VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/c4"]
CMD ["--help"]
