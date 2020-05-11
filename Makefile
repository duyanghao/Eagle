## Server version
SERVER_VERSION = v0.0.1
## Folder content generated files
BUILD_FOLDER = ./build
PROJECT_URL  = github.com/duyanghao/eagle
## command
GO           = go
GO_VENDOR    = go mod
MKDIR_P      = mkdir -p

## Random Alphanumeric String
SECRET_KEY   = $(shell cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)

## UNAME
UNAME := $(shell uname)

################################################

.PHONY: all
all: build

.PHONY: pre-build
pre-build:
	$(GO_VENDOR) vendor

.PHONY: build
build: pre-build
	$(MAKE) src.build

.PHONY: clean
	$(RM) -rf $(BUILD_FOLDER)

## src/ ########################################

.PHONY: src.build
src.build:
	cd proxy && GO111MODULE=on $(GO) build -mod=vendor -v -o ../$(BUILD_FOLDER)/proxy
	cd seeder && GO111MODULE=on $(GO) build -mod=vendor -v -o ../$(BUILD_FOLDER)/seeder

## git tag version ########################################

.PHONY: push.tag
push.tag:
	@echo "Current git tag version:"$(SERVER_VERSION)
	git tag $(SERVER_VERSION)
	git push --tags
