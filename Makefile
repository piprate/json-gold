.PHONY: all vet lint test test-cov fmt help

all: lint test

vet:
	go vet github.com/piprate/json-gold/...

test: vet
	go test github.com/piprate/json-gold/...

test-cov: vet
	go test github.com/piprate/json-gold/... -race -coverprofile=coverage.txt -covermode=atomic

lint:
	golangci-lint run

fmt:
	gofmt -s -w .

generate-report:
	SKIP_MODE=fail make test
	cp ld/earl.jsonld conformance_report.jsonld

help:
	@echo ''
	@echo ' Targets:'
	@echo '--------------------------------------------------'
	@echo ' all              - Run everything                '
	@echo ' fmt              - Format code                   '
	@echo ' lint             - Run golangci-lint             '
	@echo ' vet              - Run vet                       '
	@echo ' test             - Run all tests                 '
	@echo ' test-cov         - Run all tests + coverage      '
	@echo '--------------------------------------------------'
	@echo ''
