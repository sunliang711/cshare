# --- Build stage (skipped in CI where pre-built binaries are provided) ---
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETARCH
RUN if [ -f build/crossshare-server-linux-${TARGETARCH} ]; then \
      cp build/crossshare-server-linux-${TARGETARCH} /crossshare-server; \
    else \
      CGO_ENABLED=0 go build -ldflags="-s -w" -o /crossshare-server ./cmd/server; \
    fi

# --- Runtime stage ---
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /crossshare-server /usr/local/bin/crossshare-server
COPY config.yaml /etc/crossshare/config.yaml
EXPOSE 10431
ENTRYPOINT ["crossshare-server"]
