# Builder image
FROM golang:1.25-alpine3.23@sha256:8e02eb337d9e0ea459e041f1ee5eece41cbb61f1d83e7d883a3e2fb4862063fa AS builder
WORKDIR /workspace

ENV GO111MODULE=on

COPY go.mod go.sum ./
RUN go mod download
RUN apk add --no-cache make

COPY . .
RUN make build

# Runtime image
FROM alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11
WORKDIR /

COPY --from=builder /workspace/mqtt-prometheus-exporter .
ENTRYPOINT ["/mqtt-prometheus-exporter"]
