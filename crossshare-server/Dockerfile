# --- Frontend build stage (for standalone docker build) ---
FROM node:22-alpine AS frontend-builder
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- Go build stage (skipped in CI where pre-built binaries are provided) ---
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /web/dist ./web/dist

ARG TARGETARCH
RUN if [ -f build/crossshare-server-linux-${TARGETARCH} ]; then \
      cp build/crossshare-server-linux-${TARGETARCH} /crossshare-server; \
    else \
      CGO_ENABLED=0 go build -ldflags="-s -w" -o /crossshare-server ./cmd/server; \
    fi

# --- Runtime stage ---
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && mkdir -p /data
COPY --from=builder /crossshare-server /usr/local/bin/crossshare-server
COPY config.yaml /etc/crossshare/config.yaml
ENV CS_SERVER_DATA_DIR=/data
VOLUME ["/data"]
EXPOSE 10431
ENTRYPOINT ["crossshare-server"]
