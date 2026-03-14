# dbguard – build and development

BINARY := dbguard
PKG := ./...
CMD := ./cmd/dbguard

.PHONY: build test lint fmt clean run

build:
	go build -o $(BINARY) $(CMD)

test:
	go test $(PKG) -v -count=1

lint:
	go vet $(PKG)
	staticcheck ./... 2>/dev/null || true

fmt:
	go fmt $(PKG)
	goimports -w . 2>/dev/null || true

clean:
	rm -f $(BINARY)

run: build
	./$(BINARY) $(ARGS)

# Install binary to $GOPATH/bin or $GOBIN
install:
	go install $(CMD)
