# ========== Build Stage ==========
FROM golang:1.23-alpine AS builder

ARG VERSION=dev
ARG MEDIAMTX_VERSION=1.11.3

WORKDIR /build

COPY server-go/go.mod server-go/go.sum ./
RUN go mod download

COPY server-go/ .

RUN CGO_ENABLED=0 go build -ldflags "-X main.Version=${VERSION}" -o suisui .

# Download MediaMTX
RUN wget -q https://github.com/bluenviron/mediamtx/releases/download/v${MEDIAMTX_VERSION}/mediamtx_${MEDIAMTX_VERSION}_linux_amd64.tar.gz \
    -O mediamtx.tar.gz && tar xzf mediamtx.tar.gz && mv mediamtx /build/mediamtx && rm mediamtx.tar.gz

# ========== Run Stage ==========
FROM alpine:latest

RUN apk --no-cache add ca-certificates

# 二进制放在 /app，不和持久化卷冲突
WORKDIR /app
COPY --from=builder /build/suisui .
COPY --from=builder /build/mediamtx .

# MediaMTX 需要可执行权限
RUN chmod +x mediamtx

EXPOSE 3742 80 443 1935 8888 8889

VOLUME ["/data"]

ENV MEDIAMTX_PATH=/app/mediamtx

CMD ["./suisui", "-data", "/data"]
