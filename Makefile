.PHONY: fmt vet test build check clean

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test -race ./...

build:
	mkdir -p bin
	go build -o bin/ambient ./cmd/ambient

check: fmt vet test build

clean:
	rm -rf bin
