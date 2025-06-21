.PHONY: build
build:
	go build -o apiserver-bin -v ./apiserver

.PHONY: test
test:
	go test -v -race -timeout 30s ./...

.DEFAULT_GOAL := build