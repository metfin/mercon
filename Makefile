.PHONY: build run test clean download tidy

# Default Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run

# Binary output
BINARY_NAME=mercon
BUILD_DIR=build

# Main
MAIN_PATH=cmd/mercon/main.go

all: clean build

build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

run:
	$(GORUN) $(MAIN_PATH)

simple_run:
	$(GORUN) cmd/simple_run/main.go -wallet CSdRLr6SWaCrjCgSyJ4mSHAg3EwUzymtLT6e47uj5eX9
	
test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

download:
	$(GOMOD) download

tidy:
	$(GOMOD) tidy 