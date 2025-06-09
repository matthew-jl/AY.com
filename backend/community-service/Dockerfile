FROM golang:1.23 AS builder
WORKDIR /app
COPY user-service/ user-service/
COPY media-service/ media-service/
WORKDIR /app/community-service
COPY community-service/go.mod community-service/go.sum ./
RUN go mod download
COPY community-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o community-service ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/community-service/community-service .
EXPOSE 50057
CMD ["./community-service"]