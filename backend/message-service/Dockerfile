FROM golang:1.23 AS builder
WORKDIR /app
COPY user-service/ user-service/
COPY media-service/ media-service/
WORKDIR /app/message-service
COPY message-service/go.mod message-service/go.sum ./
RUN go mod download
COPY message-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o message-service ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/message-service/message-service .
EXPOSE 50056
CMD ["./message-service"]