FROM golang:1.23 AS builder
WORKDIR /app
COPY user-service/ user-service/
WORKDIR /app/notification-service
COPY notification-service/go.mod notification-service/go.sum ./
RUN go mod download
COPY notification-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-service ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/notification-service/notification-service .
EXPOSE 50055
CMD ["./notification-service"]