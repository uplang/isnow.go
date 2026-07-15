.DEFAULT_GOAL := help

# Bespoke gate (like the sibling up.go): the shared tools.repository Makefile
# runs `go vet ./...` and per-package coverage that cannot exclude the committed,
# machine-generated ANTLR parser under internal/isnowgrammar/. This Makefile runs
# the identical gate over the owned packages only. See .standards.yaml and
# specs/decisions/003-generated-tree.md (in the grammar repo).
#
# Quality tooling resolves from $(GOBIN) only (fleet standard: no go.mod tool
# stanza). Locally: install via nicerobot/tools.build go-tooling; in CI the
# managed go gate image bakes the same pinned set.
GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif

# The conformance corpus is the cross-implementation oracle; it lives in the
# sibling grammar repo (uplang/isnow). Locally that is the checked-out sibling;
# in CI — a lone checkout of this repo — it is absent, so fetch the public
# grammar repo once so the conformance suite runs and coverage is complete.
CORPUS := ../isnow/conformance

$(CORPUS):
	git clone --depth 1 https://github.com/uplang/isnow ../isnow

# Owned packages and files: everything except the generated parser tree.
OWNED_PKGS := $(shell go list ./... | grep -v /internal/isnowgrammar)
GO_SRC     := $(shell find . -name '*.go' -not -path './internal/isnowgrammar/*' -not -name '*_test.go')
ALL_GO     := $(shell find . -name '*.go' -not -path './internal/isnowgrammar/*')
COVERPKG   := $(shell echo $(OWNED_PKGS) | tr ' ' ',')

.PHONY: help grammars check ci fmt vet staticcheck gocognit vuln cover test build release-check

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*## "}{printf "  %-14s %s\n", $$1, $$2}'

ci: check test build ## Full gate as run by the managed CI workflow

grammars: ## Regenerate internal/isnowgrammar/ from ../isnow (needs Docker)
	$(MAKE) -C ../isnow go

check: fmt vet staticcheck gocognit vuln cover release-check ## Full quality gate

test: $(CORPUS) ## Run the conformance corpus and unit suites
	go test $(OWNED_PKGS)

build: ## Build the isnow binary
	CGO_ENABLED=0 go build -o bin/isnow ./cmd/isnow

fmt: ## gofumpt formatting (fails on any diff)
	@out="$$($(GOBIN)/gofumpt -l $(ALL_GO))"; test -z "$$out" || { echo "unformatted: $$out"; exit 1; }

vet: ## go vet (owned packages; generated tree's diagnostics filtered out)
	@go mod download  # pre-fetch so `go: downloading` notes don't pollute the captured vet output in a cold CI cache
	@out="$$(go vet $(OWNED_PKGS) 2>&1 | grep -v 'internal/isnowgrammar/' || true)"; test -z "$$out" || { echo "$$out"; exit 1; }

staticcheck: ## staticcheck (owned packages)
	$(GOBIN)/staticcheck $(OWNED_PKGS)

gocognit: ## cognitive complexity <= 7 (production files)
	@out="$$($(GOBIN)/gocognit -over 7 $(GO_SRC))"; test -z "$$out" || { echo "$$out"; exit 1; }

vuln: ## govulncheck
	$(GOBIN)/govulncheck $(OWNED_PKGS)

cover: $(CORPUS) ## tests with 100% statement coverage of owned packages
	go test -covermode=set -coverpkg=$(COVERPKG) -coverprofile=cover.out $(OWNED_PKGS)
	@go tool cover -func=cover.out | grep -q '^total:.*100.0%' || { go tool cover -func=cover.out | tail -1; echo "coverage below 100%"; exit 1; }

release-check: ## Validate the goreleaser config
	$(GOBIN)/goreleaser check
