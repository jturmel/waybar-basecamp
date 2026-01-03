BINARY_NAME=waybar-basecamp
GO_FILES=main.go

.PHONY: all build clean

all: build

build:
	go build -v -o $(BINARY_NAME) $(GO_FILES)

clean:
	rm -f $(BINARY_NAME)
