# Stage 1 - Build with Alpine-based Go image
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/auditlog ./internal/auditlog/cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/auditlog .

ENTRYPOINT ["/app/auditlog"]