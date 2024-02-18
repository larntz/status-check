FROM golang:1.21 AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o /status ./cmd/main.go
RUN pwd && ls -R

FROM scratch
COPY --from=builder /status /status
ENTRYPOINT ["/status","worker"]
