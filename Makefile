.PHONY: all vet lint test fmt help

all: lint test

vet:
	go vet github.com/piprate/json-gold/...

test: vet
	go test github.com/piprate/json-gold/...

lint:
	golangci-lint run

fmt:
	gofmt -s -w .

help:
	@echo ''
	@echo ' Targets:'
	@echo '--------------------------------------------------'
	@echo ' all              - Run everything                '
	@echo ' fmt              - Format code                   '
	@echo ' lint             - Run golangci-lint             '
	@echo ' vet              - Run vet                       '
	@echo ' test             - Run all tests                 '
	@echo '--------------------------------------------------'
	@echo ''
