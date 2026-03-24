FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /health-hub ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /health-hub-mcp ./cmd/mcp

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /health-hub /usr/local/bin/health-hub
COPY --from=builder /health-hub-mcp /usr/local/bin/health-hub-mcp

EXPOSE 8080 8081
USER 65534:65534
ENTRYPOINT ["health-hub"]
