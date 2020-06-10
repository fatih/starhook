all: test

test:
	@echo "==> running tests ..."
	@go test -cover -race  ./...

test-db:
	@echo "==> running db tests ..."
	@go test -cover -race -tags="db" ./...

install:
	@go install github.com/fatih/starhook/cmd/starhook

.PHONY: all test test-db install
