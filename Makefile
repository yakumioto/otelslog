.PHONY: test lint

test:
	go test -race -count=1 ./...

lint:
	golangci-lint run ./...