.PHONY: run build test lint docker

run:
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

test:
	go test -v -race -count=1 ./...

lint:
	golangci-lint run ./...

docker:
	docker build -t rss-aggregator .
	docker run -p 8080:8080 rss-aggregator
