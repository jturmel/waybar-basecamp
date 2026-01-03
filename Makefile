BINARY_NAME=waybar-basecamp
GO_FILES=main.go

.PHONY: all build clean setup-hooks

all: build

setup-hooks:
	git config core.hooksPath .githooks

build:
	go build -v -o $(BINARY_NAME) $(GO_FILES)

clean:
	rm -f $(BINARY_NAME)
