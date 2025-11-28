.PHONY: build test lint run clean
build:
	go build -o build/app cmd/app/main.go
test:
	go test -v -race -coverprofile=coverage.out ./...
lint:
	golangci-lint run
run:
	go run cmd/app/main.go
clean:
	rm -rf build/
