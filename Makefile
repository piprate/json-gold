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
	@echo '--------------------------------------------------'
	@echo ''
