
all: fmt lint cover

fmt:
	go fmt ./...

proto:
	(cd proto && \
	protoc -I . \
	--go_out=plugins=grpc:. \
	simple.proto)

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
	go get -u ./...
	go mod tidy

tools:
	go install golang.org/x/tools/cmd/cover@latest
	go install golang.org/x/lint/golint@latest

.PHONY: all fmt lint vet test cover build clean ensure update tools proto