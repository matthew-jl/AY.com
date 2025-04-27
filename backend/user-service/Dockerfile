FROM golang:1.23 AS builder
WORKDIR /app
COPY backend/user-service/go.mod backend/user-service/go.sum ./
# Perlu karena ada di replace directive di go.mod, biar go mod download tidak error
COPY backend/genproto /genproto
RUN go mod download
# Destination: ./ relative to WORKDIR (/app).
COPY backend/user-service/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o user-service .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/user-service .
EXPOSE 50051
CMD ["./user-service"]