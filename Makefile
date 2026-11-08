.PHONY: all
all: test

.PHONY: fmt
fmt:
	go vet ./...
	go fix ./...
	go fmt ./...

.PHONY: clean
clean:
	rm -f coverprofile.out

.PHONY: test
test: fmt
	golangci-lint run ./...
	ginkgo run -cover ./...
