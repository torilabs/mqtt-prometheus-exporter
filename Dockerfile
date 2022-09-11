# Builder image
FROM golang:1.19 as builder
WORKDIR /workspace

ENV GO111MODULE=on

COPY go.mod go.sum ./
RUN go mod download

ADD . .
RUN make build

# Runtime image
FROM alpine:3.16.0
WORKDIR /

COPY --from=builder /workspace/mqtt-prometheus-exporter .
ENTRYPOINT ["/mqtt-prometheus-exporter"]
