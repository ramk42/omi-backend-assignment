FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/account ./internal/account/cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/account .

EXPOSE 8080

ENTRYPOINT ["/app/account"]