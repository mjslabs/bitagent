PKG := $(shell head -1 go.mod | sed -e 's/module //')
OUT := $(shell basename ${PKG})
VERSION := $(shell git describe --tag --long --dirty)
PKG_LIST := $(shell go list ${PKG}/...)
GO_FILES := $(shell find . -name '*.go')

all: build

.PHONY: build
build:
	go build -v -o ${OUT} ${PKG}

c.out:
	go test -coverprofile=c.out -v ${PKG}/...

.PHONY: test
test: c.out

cover.html: c.out
	go tool cover -html=c.out -o cover.html

.PHONY: vet
vet:
	@go vet ${PKG_LIST}

.PHONY: lint
lint:
	@for file in ${GO_FILES} ; do \
		golint $$file ; \
	done

.PHONY: static
static: vet lint
	go build -v -o ${OUT}-${VERSION} -a \
		-tags netgo \
		-gcflags=all=-trimpath=${GOPATH} \
		-asmflags=all=-trimpath=${GOPATH} \
		-ldflags="-extldflags \"-static\" -w -s" \
		${PKG}

.PHONY: clean
clean:
	rm -f ${OUT} ${OUT}-v* c.out cover.html
