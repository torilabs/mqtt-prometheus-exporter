SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*" )
export GO111MODULE := on
export CGO_ENABLED := 0

EXECUTABLE = mqtt-prometheus-exporter
INTEGRATION_TEST_PATH ?= ./it

all: clean check test build

MAKEFLAGS += --no-print-directory

prepare:
	@echo "Downloading tools"
ifeq (, $(shell which go-junit-report))
	go install github.com/jstemmer/go-junit-report@latest
endif
ifeq (, $(shell which gocov))
	go install github.com/axw/gocov/gocov@latest
endif
ifeq (, $(shell which gocov-xml))
	go install github.com/AlekSi/gocov-xml@latest
endif

check:
	@echo "Running check"
ifeq (, $(shell which golangci-lint))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.49.0
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

docker.start.components:
	docker-compose --file $(INTEGRATION_TEST_PATH)/docker-compose.yml up -d --remove-orphans mosquitto

test.integration: docker.start.components
	go test -tags=integration $(INTEGRATION_TEST_PATH) -count=1 -v
	@$(MAKE) --no-print-directory docker.stop

docker.stop:
	docker-compose --file $(INTEGRATION_TEST_PATH)/docker-compose.yml down

build:
	@echo "Running build"
	go build -v -o "$(EXECUTABLE)"

clean:
	@echo "Running clean"
	rm -rf report docs tmp
	rm -f $(EXECUTABLE)
