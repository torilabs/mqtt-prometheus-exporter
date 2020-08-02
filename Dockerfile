# Builder image
FROM golang:1.14 as builder
WORKDIR /workspace

ENV GO111MODULE=on

COPY go.mod go.sum ./
RUN go mod download

ADD . .
RUN make build

# Runtime image
FROM alpine:3.11.3
WORKDIR /

COPY --from=builder /workspace/mqtt-prometheus-exporter .
COPY --from=builder /workspace/config.yaml .
ENTRYPOINT ["/mqtt-prometheus-exporter"]
