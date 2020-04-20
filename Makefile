PKG := $(shell head -1 go.mod | sed -e 's/module //')
OUT := $(shell basename ${PKG})
VERSION := $(shell git describe --tag --long --dirty)
PKG_LIST := $(shell go list ${PKG}/...)
GO_FILES := $(shell find . -name '*.go')

all: build

build:
	go build -v -o ${OUT} ${PKG}/cmd/bitagent

c.out:
	go test -coverprofile=c.out -v ${PKG}/...

test: c.out

coverhtml: c.out
	go tool cover -html=c.out

vet:
	@go vet ${PKG_LIST}

lint:
	@for file in ${GO_FILES} ; do \
		golint $$file ; \
	done

static: vet lint
	go build -v -o ${OUT}-${VERSION} -a \
		-tags netgo \
		-gcflags=all=-trimpath=${GOPATH} \
		-asmflags=all=-trimpath=${GOPATH} \
		-ldflags="-extldflags \"-static\" -w -s" \
		${PKG}

clean:
	rm -f ${OUT} ${OUT}-v* c.out

.PHONY: build test coverhtml vet lint static clean
