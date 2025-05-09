FROM golang:1.23 AS builder
WORKDIR /app
COPY user-service/ user-service/
COPY media-service/ media-service/
COPY thread-service/ thread-service/
WORKDIR /app/api-gateway
COPY api-gateway/go.mod api-gateway/go.sum ./
RUN go mod download
COPY api-gateway/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/api-gateway/api-gateway .
EXPOSE 8080
CMD ["./api-gateway"]