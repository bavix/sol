.PHONY: *

test:
	go test -tags mock -race -cover ./...

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.1 run --color always ${args}

lint-fix:
	make lint args=--fix
