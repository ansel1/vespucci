
all: fmt lint cover

fmt:
	(cd v4 && go fmt ./...)

lint:
	(cd v4 && golint ./...)

vet:
	(cd v4 && go vet ./...)

test:
	(cd v4 && go test ./...)


cover:
	if [ ! -d build ]; then mkdir build; fi
	(cd v4 && go test ./... -covermode=count -coverprofile=../build/coverage.out)
	(cd v4 && go tool cover -html=../build/coverage.out -o ../build/coverage.html)

build:
	(cd v4 && go build ./...)

clean:
	rm -rf build/
	go clean

update:
	(cd v4 && go get -u)

tools:
	go get -u golang.org/x/tools/cmd/cover
	go get -u golang.org/x/lint/golint

.PHONY: all fmt lint vet test cover build clean ensure update tools