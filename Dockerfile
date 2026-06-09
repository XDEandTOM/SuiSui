# ========== Build Stage ==========
FROM golang:1.23-alpine AS builder

WORKDIR /build

COPY server-go/go.mod server-go/go.sum ./
RUN go mod download

COPY server-go/ .

RUN CGO_ENABLED=0 go build -o suisui .

# ========== Run Stage ==========
FROM alpine:latest

RUN apk --no-cache add ca-certificates

# 二进制放在 /app，不和持久化卷冲突
WORKDIR /app
COPY --from=builder /build/suisui .

EXPOSE 3001

VOLUME ["/data"]

CMD ["./suisui", "-data", "/data"]
