CC = go
NAME ?= ginctl
export PACKAGE_NAME ?= $$CI_PROJECT_NAME
REVISION := $(shell git rev-list --tags --max-count=1)
VERSION := $(shell git describe --tags $(REVISION))

PKG = $(PACKAGE_NAME)

GOX = gox

GO_BUILD_LDFLAGS="-s -w -X main.version=$(VERSION)"

install:
	$(CC) build -o $(NAME) main.go && mv $(NAME) /usr/local/bin

$(GOX):
	go get github.com/mitchellh/gox

build: $(GOX)
	gox $(BUILD_PLATFORMS) \
		-output="out/binaries/$(PACKAGE_NAME)-{{.OS}}-{{.Arch}}" \
		-ldflags=$(GO_BUILD_LDFLAGS) \
		$(PKG)

release_upload:
	@./ci/release_upload "$$CI_COMMIT_TAG"

.PHONY: install build release_upload $(GOX)
