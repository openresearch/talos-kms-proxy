GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
BINARY_NAME=orkms
VERSION=$(shell git describe --always --tags)
COMMIT=$(shell git rev-parse --verify HEAD)
BUILD_DATE=$(shell date -u +'%F %T %z %Z')

ifeq ($(BIN),)
BIN := $(shell pwd)/bin
endif

all: deps build
local: depslocal build
pkg: depspkg build_binary


test:
	golangci-lint run

build:
	mkdir -p bin
	cd cmd && CGO_ENABLED=0 $(GOBUILD) \
		-ldflags  "-s -w -X 'main.version=${VERSION}' \
		-X 'main.commit=${COMMIT}' \
		-X 'main.date=${BUILD_DATE}' \
		-X 'main.appname=${BINARY_NAME}'" \
		-o $(BIN)/$(BINARY_NAME) && cd ..
depslocal:
	go mod tidy -v
clean: 
	rm -f $(BIN)/$(BINARY_NAME)
	go clean -modcache
