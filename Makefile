BIN := silverjoy

GOPATH := $(shell go env GOPATH)

ifeq ($(OS),Windows_NT)
    GO_FILES := $(shell dir /S /B *.go)
    GO_DEPS := $(shell dir /S /B go.mod go.sum)
    CLEAN := del
else
    GO_FILES := $(shell find . -name '*.go')
    GO_DEPS := $(shell find . -name go.mod -o -name go.sum)
    CLEAN := rm -f
endif

$(BIN): $(GO_FILES) $(GO_DEPS)
	$(MAKE) pretty
	go vet ./...
	go build -o $(BIN) main.go

.PHONY: test
test: $(BIN)
	./$(BIN) --verbose

.PHONY: pretty
pretty: $(GO_FILES)
	gofumpt -w $^

.PHONY: install
install: $(GOPATH)/bin/$(BIN)

$(GOPATH)/bin/$(BIN): $(BIN)
	mv $(BIN) $(GOPATH)/bin/$(BIN)

.PHONY: clean
clean:
	$(CLEAN) $(BIN)
