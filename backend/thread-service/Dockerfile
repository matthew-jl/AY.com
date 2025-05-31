FROM golang:1.23 AS builder
WORKDIR /app
COPY user-service/ user-service/
COPY search-service/ search-service/
WORKDIR /app/thread-service
COPY thread-service/go.mod thread-service/go.sum ./
RUN go mod download
COPY thread-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o thread-service ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/thread-service/thread-service .
EXPOSE 50052
CMD ["./thread-service"]