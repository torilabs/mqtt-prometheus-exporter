# Builder image
FROM golang:1.24.8 AS builder
WORKDIR /workspace

ENV GO111MODULE=on

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

# Runtime image
FROM alpine:3.22.1
WORKDIR /

COPY --from=builder /workspace/mqtt-prometheus-exporter .
ENTRYPOINT ["/mqtt-prometheus-exporter"]
