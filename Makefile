export GO111MODULE=on
export GOVCS=*:git

TARGET   = imk
MAIN     = ./cmd/main.go
BIN      = ./bin
COVEROUT = cover.out

all: build
BRANCH   := ${shell git rev-parse --abbrev-ref HEAD}
REVCNT   := ${shell git rev-list --count $(BRANCH)}
REVHASH  := ${shell git log -1 --format="%h"}
LDFLAGS  := -X main.version=${BRANCH}.${REVCNT}.${REVHASH}
CFLAGS   := --ldflags '${LDFLAGS}' -o $(BIN)/$(TARGET)

lint:
	golangci-lint run

install_deps_tools:
	go install github.com/matryer/moq@latest

cover:
	go tool cover -html=$(COVEROUT)
	-rm -f $(COVEROUT)

test:
	go test -timeout $(TIMEOUT)s -cover -coverprofile=$(COVEROUT) ./pkg/...

# requires moq tool to be installed
# go install github.com/matryer/moq@latest
generate:
	go generate ./internal/...

build: clean
	go build ${CFLAGS} $(MAIN)

build-linux: clean
	CGO_ENABLED=0 GOOS=linux go build ${CFLAGS} -a -installsuffix cgo $(MAIN)

clean:
	-rm -rf $(BIN)
	-rm -f $(COVEROUT)

.PHONY: build build-linux
