.PHONY: all lint test fmt help

all: lint test

lint:
	go vet github.com/piprate/json-gold/...

test: lint
	go test github.com/piprate/json-gold/...

fmt:
	gofmt -s -w .

help:
	@echo ''
	@echo ' Targets:'
	@echo '--------------------------------------------------'
	@echo ' all              - Run everything                '
	@echo ' fmt              - Format code                   '
	@echo ' lint             - Run lint                      '
	@echo ' test             - Run all tests                 '
	@echo '--------------------------------------------------'
	@echo ''
