# 对象存储构建：多阶段编译 Go API 为静态二进制，兼容 linux/amd64 与 linux/arm64。
FROM --platform=$BUILDPLATFORM golang:1.22 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/api ./cmd/api

FROM --platform=$TARGETPLATFORM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /out/api /app/api
ENV HTTP_ADDR=:8080
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=20s --retries=6 \
  CMD curl -fsS http://localhost:8080/health || exit 1
ENTRYPOINT ["/app/api"]
