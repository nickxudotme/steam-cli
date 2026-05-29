GO ?= go
BIN ?= steam-cli
APPID ?= 264710
STEAM_CC ?= US
STEAM_LANG ?= english

.DEFAULT_GOAL := help

.PHONY: help build vet test test-race check fmt clean doctor smoke

help:
	@printf '%s\n' \
		'Targets:' \
		'  build      Build ./$(BIN)' \
		'  vet        Run go vet ./...' \
		'  test       Run go test ./...' \
		'  test-race  Run go test -race ./...' \
		'  check      Run vet, race tests, and build' \
		'  fmt        Run gofmt on Go files' \
		'  clean      Remove ./$(BIN)' \
		'  doctor     Build and run ./$(BIN) doctor' \
		'  smoke      Build and run a live price smoke check'

build:
	$(GO) build -o $(BIN) .

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

check: vet test-race build

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

clean:
	rm -f $(BIN)

doctor: build
	./$(BIN) doctor

smoke: build
	./$(BIN) price $(APPID) --cc $(STEAM_CC) --lang $(STEAM_LANG)
