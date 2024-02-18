FROM golang:1.21 AS builder

WORKDIR /app
COPY . .
RUN go mod download && CGO_ENABLED=0 go build -o /status ./cmd/main.go

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /status /status
ENTRYPOINT ["/status","worker"]
