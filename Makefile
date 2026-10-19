SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*" )
export GO111MODULE := on
export CGO_ENABLED := 0

EXECUTABLE = mqtt-prometheus-exporter
# version reported at startup: taken from the environment (e.g. docker build arg), else from git
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -X github.com/torilabs/mqtt-prometheus-exporter/version.Version=$(VERSION)
INTEGRATION_TEST_PATH ?= ./it

all: clean check test build

MAKEFLAGS += --no-print-directory

prepare:
	@echo "Downloading tools"
ifeq (, $(shell which go-junit-report))
	go install github.com/jstemmer/go-junit-report@latest
endif

check:
	@echo "Running check"
ifeq (, $(shell which golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v2.10.1
endif
	golangci-lint run
	go mod tidy

test: prepare
	@echo "Running tests"
	mkdir -p report
	export CGO_ENABLED=1 && go test -race -v ./... -coverprofile=report/coverage.txt | tee report/report.txt
	go-junit-report -set-exit-code < report/report.txt > report/report.xml
	go mod tidy

build:
	@echo "Running build"
	go build -v -ldflags "$(LDFLAGS)" -o "$(EXECUTABLE)"

clean:
	@echo "Running clean"
	rm -rf report docs tmp
	rm -f $(EXECUTABLE)
