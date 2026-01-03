BINARY_NAME=waybar-basecamp
GO_FILES=main.go

.PHONY: all build clean setup-hooks

all: build

setup-hooks:
	git config core.hooksPath .githooks

build:
	CGO_ENABLED=1 go build -v -tags netgo,osusergo -ldflags '-extldflags "-static"' -o $(BINARY_NAME) $(GO_FILES)

clean:
	rm -f $(BINARY_NAME)
