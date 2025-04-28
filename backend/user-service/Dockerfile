FROM golang:1.23 AS builder
WORKDIR /app/user-service
COPY user-service/go.mod user-service/go.sum ./
RUN go mod download
COPY user-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o user-service ./cmd

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/user-service/user-service .
EXPOSE 50051
CMD ["./user-service"]