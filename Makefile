SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*" )
export GO111MODULE := on
export CGO_ENABLED := 0

EXECUTABLE = mqtt-prometheus-exporter

all: clean check test build

MAKEFLAGS += --no-print-directory

prepare:
	@echo "Downloading tools"
ifeq (, $(shell which go-junit-report))
	go get github.com/jstemmer/go-junit-report
endif
ifeq (, $(shell which gocov))
	go get github.com/axw/gocov/gocov
endif
ifeq (, $(shell which gocov-xml))
	go get github.com/AlekSi/gocov-xml
endif

check:
	@echo "Running check"
ifeq (, $(shell which golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $(GOPATH)/bin v1.30.0
endif
	golangci-lint run
	go mod tidy

test: prepare
	@echo "Running tests"
	mkdir -p report
	export CGO_ENABLED=1 && go test -race -v ./... -coverprofile=report/coverage.txt | tee report/report.txt
	go-junit-report -set-exit-code < report/report.txt > report/report.xml
	gocov convert report/coverage.txt | gocov-xml > report/coverage.xml
	go mod tidy

build:
	@echo "Running build"
	go build -v -o "$(EXECUTABLE)"

clean:
	@echo "Running clean"
	rm -rf report docs tmp
	rm -f $(EXECUTABLE)
