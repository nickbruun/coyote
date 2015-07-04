##
# Project configuration.
#
# Packages for binaries is expected to be located in
# src/$(PROJECT)/bin/$(BINARY). By default, distributions are built for i386
# and AMD64 on Linux as well as AMD64 on Darwin.
REPOSITORY := github.com/nickbruun/coyote
PACKAGES := \
	output \
	utils
BINARIES := coyoterun
LIBRARIES :=


##
# Utilities.
SPACE = $(noop) $(noop)

##
# Build directory.
BUILD_DIR := build
BUILD_SRC_DIR := $(BUILD_DIR)/src
BUILD_REPOSITORY_PATH := $(BUILD_SRC_DIR)/$(REPOSITORY)
BUILD_REPOSITORY_PARENT_DIR := $(dir $(BUILD_REPOSITORY_PATH))

##
# Source and destination files.
override PACKAGES := $(PACKAGES) $(BINARIES:%=bin/%)

SOURCE := $(addsuffix /*.go, $(addprefix $(BUILD_SRC_DIR)/$(REPOSITORY)/, $(PACKAGES)))
LIBRARIES_DIRS := $(addprefix $(REPOSITORY)/src/, $(LIBRARIES))
BINARY_PATHS := $(addprefix $(BUILD_DIR)/bin/, $(BINARIES))
DIST_PREFIXED_BINARY_PATHS := $(addprefix dist/, $(BINARIES))
DIST_LINUX_AMD64_BINARY_PATHS := \
	$(addsuffix -linux-amd64, $(DIST_PREFIXED_BINARY_PATHS))
DIST_LINUX_386_BINARY_PATHS := \
	$(addsuffix -linux-386, $(DIST_PREFIXED_BINARY_PATHS))
DIST_DARWIN_AMD64_BINARY_PATHS := \
	$(addsuffix -darwin-amd64, $(DIST_PREFIXED_BINARY_PATHS))
DIST_BINARY_PATHS := \
	$(DIST_LINUX_AMD64_BINARY_PATHS) \
	$(DIST_LINUX_386_BINARY_PATHS) \
	$(DIST_DARWIN_AMD64_BINARY_PATHS)

##
# Installation paths.
PREFIX ?= /usr/local
INSTALL_PREFIX := $(patsubst %/, %, $(PREFIX))
INSTALLED_BINARY_PATHS = $(addprefix $(INSTALL_PREFIX)/, $(BINARY_PATHS))

##
# Canned recipes.
dist-binary-name=$(1:dist/%=%)
dist-binary-components=$(subst -, ,$(call dist-binary-name,$1))
dist-binary-bin=$(word 1,$(call dist-binary-components,$1))
dist-binary-os=$(word 2,$(call dist-binary-components,$1))
dist-binary-arch=$(word 3,$(call dist-binary-components,$1))

# Build distribution.
#
# Must be invoked with the target distribution binary path as
# dist/$(BINARY_NAME)-$(OS)-$(ARCH)
define build-dist
	CGO_ENABLED=0 \
	GOARCH=$(call dist-binary-arch,$@) \
	GOOS=$(call dist-binary-os,$@) \
	go build -v -o $@ $(PROJECT)/bin/$(call dist-binary-bin,$@)
endef

##
# Build targets.
export GOPATH=$(shell pwd)/$(BUILD_DIR)

all: $(BINARY_PATHS)

$(SOURCE) $(LIBRARIES_DIRS): $(BUILD_REPOSITORY_PATH)

$(BUILD_REPOSITORY_PARENT_DIR):
	mkdir -p $@

$(BUILD_REPOSITORY_PATH): $(BUILD_REPOSITORY_PARENT_DIR)
	ln -Fs $(subst $(SPACE),/,$(patsubst %,..,$(subst /, ,$(dir $(BUILD_REPOSITORY_PATH))))) $(BUILD_REPOSITORY_PATH)

$(BUILD_DIR)/bin/%: $(SOURCE) $(LIBRARIES_DIRS)
	go build -v -o $@ $(REPOSITORY)/$(@:$(BUILD_DIR)/%=%)

format:
	gofmt -l -w $(SOURCE)

$(LIBRARIES_DIRS): $(BUILD_REPOSITORY_PATH)
	go get $(@:$(BUILD_SRC_DIR)/%=%)

test: $(SOURCE) $(LIBRARIES_DIRS)
	go test -v $(addprefix $(REPOSITORY)/,$(PACKAGES))

dist/%: $(SOURCE) $(LIBRARIES_DIRS)
	$(build-dist)

dist: $(DIST_BINARY_PATHS)

clean:
	rm -rf $(BUILD_DIR)

$(INSTALL_PREFIX)/bin/%: $(BINARY_PATHS)
	install $(@:$(INSTALL_PREFIX)/%=%) $@

install: $(INSTALLED_BINARY_PATHS)

.PHONY: test clean format dist install
