.PHONY: build
build: update
	go build ./...

.PHONY: test
test: update
	go test ./...

.PHONY: update
update:
	go mod tidy

.PHONY: lint
lint:
	golangci-lint run -c .golangci.yaml
