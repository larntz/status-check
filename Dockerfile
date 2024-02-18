FROM golang:1.21 AS builder

WORKDIR /app
COPY . .
RUN ls -R
RUN go mod download
RUN make build

FROM scratch

COPY --from=builder /app/status /bin/status

ENTRYPOINT ["/bin/status"]
