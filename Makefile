.PHONY: setup
setup:
	go mod download
	go tool lefthook install
	@echo "✅ Development environment setup complete"

.PHONY: test
test:
	go test -cover ./...

.PHONY: lint
lint:
	go tool golangci-lint run

.PHONY: build
build:
	go build .
	@echo "✅ Production binary built (without dev tools)"
