
all: fmt lint cover

fmt:
	go fmt ./...

lint:
	golint ./...

vet:
	go vet ./...

test:
	go test ./...


cover:
	if [ ! -d build ]; then mkdir build; fi
	go test ./... -covermode=count -coverprofile=build/coverage.out
	go tool cover -html=build/coverage.out -o build/coverage.html

build:
	go build ./...

clean:
	rm -rf build/
	go clean

update:
	go get -u

tools:
	go get -u golang.org/x/tools/cmd/cover
	go get -u golang.org/x/lint/golint

.PHONY: all fmt lint vet test cover build clean ensure update tools