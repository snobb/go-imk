export GO111MODULE=on
export GOVCS=*:git

TARGET   = imk
MAIN     = ./cmd/main.go
BIN      = ./bin
COVEROUT = cover.out

VERSION  := $(shell git describe --tags --abbrev=1 --always --dirty=-dev)
LDFLAGS  := -X main.version=${VERSION} -s
CFLAGS   := --ldflags '${LDFLAGS}' -o $(BIN)/$(TARGET)
TIMEOUT  := 5

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}' ${MAKEFILE_LIST}

.PHONY: install_deps_tools
install_deps_tools: ## Install dev dependencies
	go install github.com/matryer/moq@latest

.PHONY: init-hooks
init-hooks:
	cp -f gitHooks/* .git/hooks/
	chmod +x .git/hooks/*

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: cover
cover: ## Run coverage
	go tool cover -html=$(COVEROUT)
	-rm -f $(COVEROUT)

.PHONY: test
test: ## Run tests
	go test -timeout $(TIMEOUT)s -cover -coverprofile=$(COVEROUT) ./internal/...

# requires moq tool to be installed
# go install github.com/matryer/moq@latest
.PHONY: generate
generate: ## Generate mocks
	go generate ./internal/...

.PHONY: build
build: clean ## Build binary
	CGO_ENABLED=0 go build ${CFLAGS} $(MAIN)

.PHONY: build-linux
build-linux: clean ## Build linux binary
	CGO_ENABLED=0 GOOS=linux go build ${CFLAGS} -a -installsuffix cgo $(MAIN)

.PHONY: clean ## Clean everything
clean:
	-rm -rf $(BIN)
	-rm -f $(COVEROUT)
