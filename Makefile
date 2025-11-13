# Config
BINARY      := rats
GO          ?= go
LINTER      ?= golangci-lint
ALIGNER     ?= betteralign
BENCHSTAT   ?= benchstat
SBOM        ?= cyclonedx-gomod

# Toolchain flags
CGO_ENABLED ?= 0
GOFLAGS     ?= -buildvcs=false -trimpath
LDFLAGS     ?= -s -w
GOFTAGS     ?= forceposix

PKG         := github.com/woozymasta/rats
CMD_DIR     := ./cmd/$(BINARY)

# Host env
GOOS        ?= $(shell $(GO) env GOOS)
GOARCH      ?= $(shell $(GO) env GOARCH)
SUFFIX      := $(if $(filter $(GOOS),windows),.exe,)

# Bench output
BENCH_DIR   := bench
BENCH_TS    := $(shell date +%Y%m%d_%H%M%S)
GIT_SHA     := $(shell git rev-parse --short HEAD)
BENCH_FILE  := $(BENCH_DIR)/bench_$(BENCH_TS)_$(GIT_SHA).txt

# Dist output
DIST        := dist

# Build matrix (OS x ARCH)
OS_LIST     ?= linux darwin windows
ARCH_LIST   ?= amd64 arm64

# Git hooks
HOOKS_DIR   := .githooks

# Default
all: build

# Local build for host platform (CLI) + SBOM
build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	  $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' \
			-tags '$(GOFTAGS)' -o bin/$(BINARY)$(SUFFIX) $(CMD_DIR)/
	@if command -v $(SBOM) >/dev/null 2>&1; then \
	  echo ">> SBOM bin/$(BINARY)$(SUFFIX)"; \
	  $(SBOM) bin -json -output bin/$(BINARY)$(SUFFIX).sbom.json bin/$(BINARY)$(SUFFIX) \
	    || $(SBOM) app -json -output bin/$(BINARY)$(SUFFIX).sbom.json bin/$(BINARY)$(SUFFIX); \
	else \
	  echo "!! $(SBOM) not found; skip SBOM for bin/$(BINARY)$(SUFFIX)"; \
	fi

# Install CLI into GOBIN/GOPATH/bin
install:
	$(GO) install $(PKG)/cmd/$(BINARY)@latest

# Tests (lib) + ensure CLI compiles
test:
	$(GO) test ./...
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' $(CMD_DIR)/

download:
	$(GO) mod download

fmt:
	$(GO) fmt ./...

verify:
	$(GO) mod verify

vet:
	$(GO) vet ./...

# Lint
lint:
	$(LINTER) run ./...

# Alignment check
align:
	$(ALIGNER) ./...

# go mod tidy in both modules
tidy:
	@set -e; \
		echo ">> go mod tidy"; \
		$(GO) mod tidy; \

# Validate before commit
validate: download tidy fmt verify vet test lint align
	@echo "OK"

validate-clean: download tidy fmt verify vet test lint align
	@git diff --quiet --exit-code || { \
		echo "working tree is dirty after validate"; \
		exit 1; \
	}
	@echo "OK"

# Bench log
bench-log:
	@mkdir -p $(BENCH_DIR)
	@echo "# bench $(BENCH_TS) commit $(GIT_SHA)" | tee -a $(BENCH_FILE)
	@$(GO) test -run='^$$' -bench=. -benchmem ./... | tee -a $(BENCH_FILE)
	@echo "wrote $(BENCH_FILE)"

# Bench diff (last two logs)
bench-diff:
	@set -e; \
	files=$$(ls -1 $(BENCH_DIR)/bench_*.txt 2>/dev/null | sort | tail -n 2); \
	[ -n "$$files" ] || { echo "bench-diff: no bench logs"; exit 0; }; \
	cnt=$$(echo $$files | wc -w); \
	if [ "$$cnt" -lt 2 ]; then echo "bench-diff: need at least two logs"; exit 0; fi; \
	if command -v $(BENCHSTAT) >/dev/null 2>&1; then \
		$(BENCHSTAT) $$files; \
	else \
		$(GO) run golang.org/x/perf/cmd/benchstat $$files; \
	fi

# Full bench workflow: log + diff
bench: bench-log bench-diff

# Build matrix: $(OS_LIST) x $(ARCH_LIST) into dist/
build-matrix:
	@set -e; mkdir -p $(DIST); \
	for os in $(OS_LIST); do \
	  for arch in $(ARCH_LIST); do \
	    ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
	    out="$(DIST)/$(BINARY)-$${os}-$${arch}$${ext}"; \
	    echo ">> building $${out}"; \
	    CGO_ENABLED=$(CGO_ENABLED) GOOS=$${os} GOARCH=$${arch} \
	      $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -tags '$(GOFTAGS)' \
					-o "$${out}" $(CMD_DIR)/; \
	  done; \
	done

# Generate SBOMs for all dist artifacts
sbom-dist:
	@set -e; \
	ls -1 $(DIST)/$(BINARY)-* >/dev/null 2>&1 || { echo "no artifacts in $(DIST)"; exit 0; }; \
	for f in $(DIST)/$(BINARY)-*; do \
	  [ -f "$$f" ] || continue; \
	  echo ">> SBOM $$f"; \
	  if command -v $(SBOM) >/dev/null 2>&1; then \
	    $(SBOM) bin -json -output "$$f.sbom.json" "$$f" \
	      || $(SBOM) app -json -output "$$f.sbom.json" "$$f"; \
	  else \
	    echo "!! $(SBOM) not found; skip SBOM for $$f"; \
	  fi; \
	done

# Checksums for dist artifacts
checksums:
	@set -e; \
	ls -1 $(DIST)/$(BINARY)-* 1>/dev/null 2>&1 || { echo "no artifacts in $(DIST)"; exit 0; }; \
	( cd $(DIST) && shasum -a 256 $(BINARY)-* 2>/dev/null || sha256sum $(BINARY)-* ) > $(DIST)/SHA256SUMS
	@echo "wrote $(DIST)/SHA256SUMS"

# Release bundle: build + sbom + checksums
release: clean-dist build-matrix sbom-dist checksums
	@echo "Artifacts in $(DIST)/"

# Clean
clean:
	rm -rf bin/

clean-dist:
	rm -rf $(DIST)/

# Release helpers
tag:
	@test -n "$(VERSION)" || VERSION="$(word 2,$(MAKECMDGOALS))"; \
	[ -n "$$VERSION" ] || { echo "VERSION is required"; exit 2; }; \
	git tag v$$VERSION && git push origin v$$VERSION

%:
	@:

# Tool installer
tools:
	@echo ">> installing golangci-lint"
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@echo ">> installing betteralign"
	$(GO) install github.com/dkorunic/betteralign/cmd/betteralign@latest
	@echo ">> installing benchstat"
	$(GO) install golang.org/x/perf/cmd/benchstat@latest
	@echo ">> installing cyclonedx-gomod"
	$(GO) install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest


# Enable git hooks into .githooks
hooks-enable:
	@git config core.hooksPath $(HOOKS_DIR)
	@echo "hooks installed to $(HOOKS_DIR) and enabled"

# Disable git hooks
hooks-disable:
	@git config --unset core.hooksPath || :
	@echo "hooks disabled (core.hooksPath unset)"

.PHONY: \
	all build install test fmt verify vet lint align tidy validate validate-clean \
	bench-log bench-diff bench \
	build-matrix sbom-dist checksums release \
	clean clean-dist \
	tag-lib tag-cli \
	tools \
	hooks-enable hooks-disable
