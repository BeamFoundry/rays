.PHONY: all
all: test

.PHONY: generate
generate:
	go generate ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...


.PHONY: clean
clean:
	rm -f coverprofile.out

.PHONY: test
test: generate fmt vet
	ginkgo run -cover ./...
