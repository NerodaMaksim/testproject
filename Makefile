.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test  ./...
	
.PHONY: build
build:
	go build -o bin/testproject ./cmd	