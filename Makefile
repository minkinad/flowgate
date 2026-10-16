.PHONY: setup build run test lint fmt check docker-build
setup:
	go mod download
build:
	go build -o bin/flowgate ./cmd/flowgate
run:
	go run ./cmd/flowgate serve --config configs/flowgate.yaml
test:
	go test ./...
lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0 run ./...
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')
check:
	go vet ./...
	go test -race ./...
docker-build:
	docker build -t flowgate:dev .
