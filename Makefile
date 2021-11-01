CGO_ENABLED ?= 0
LD_FLAGS ?= "-extldflags '-static'"

GO_BIN ?= go
GO_CMD ?= CGO_ENABLED=$(CGO_ENABLED) $(GO_BIN)

GO_TEST=$(GO_CMD) test -covermode=atomic
GO_MOD=$(GO_CMD) mod
GO_RUN=$(GO_CMD) run
GO_BUILD=$(GO_CMD) build -v -ldflags $(LD_FLAGS) -trimpath
GO_PACKAGES ?= ./...
GO_TESTS ?= ^.*$

GOLANGCI_LINT ?= $(GO_RUN) github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0

COVERPROFILE=c.out

.PHONY: all
all: build build-test test lint codespell semgrep

.PHONY: build
build:
	$(GO_BUILD) ./cmd/...

.PHONY: test
test: build-test ## Run unit tests.
	$(GO_TEST) -run $(GO_TESTS) $(GO_PACKAGES)

.PHONY: lint
lint: ## Run linter.
	$(GOLANGCI_LINT) run $(GO_PACKAGES)

.PHONY: codespell
codespell: CODESPELL_BIN ?= codespell
codespell: ## Runs spell checking.
	@if ! which $(CODESPELL_BIN) >/dev/null 2>&1; then echo "$(CODESPELL_BIN) binary not found, skipping spell checking"; else $(CODESPELL_BIN) --check-filenames --check-hidden --skip .git; fi

.PHONY: semgrep
semgrep: SEMGREP_BIN ?= semgrep
semgrep: ## Runs semgrep linter.
	@if ! which $(SEMGREP_BIN) >/dev/null 2>&1; then echo "$(SEMGREP_BIN) binary not found, skipping extra linting"; else $(SEMGREP_BIN); fi

.PHONY: test-mutate
test-mutate: ## Run mutation tests.
	$(GO_RUN) github.com/Stebalien/go-mutesting/cmd/go-mutesting@3245157 --verbose $(GO_PACKAGES)

.PHONY: check-test-mutate
check-test-mutate: SHELL=/bin/bash
check-test-mutate: test-working-tree-clean ## Verifies that test code reaches 100% mutation scopre.
	$(GO_RUN) github.com/Stebalien/go-mutesting/cmd/go-mutesting@3245157 --verbose $(GO_PACKAGES) | tee >(cat 1>&2) | grep 'mutation score is 1.000000' >/dev/null

.PHONY: build-test
build-test: ## Compile unit tests.
	$(GO_TEST) -run=nope $(GO_PACKAGES)

.PHONY: test-cover
test-cover: build-test ## Run unit tests with coverage file generation.
	$(GO_TEST) -run $(GO_TESTS) -coverprofile=$(COVERPROFILE) $(GO_PACKAGES)

.PHONY: test-race
test-race: CGO_ENABLED := 1
test-race: build-test ## Run unit tests with race detector.
	$(GO_TEST) -run $(GO_TESTS) -race $(GO_PACKAGES)

.PHONY: test-working-tree-clean
test-working-tree-clean: ## Check if working directory is clean.
	@test -z "$$(git status --porcelain)" || (echo "Commit all changes before running this target"; exit 1)

.PHONY: test-tidy
test-tidy: test-working-tree-clean ## Check if Go modules are tied.
	$(GO_MOD) tidy
	@test -z "$$(git status --porcelain)" || (echo "Please run '$(GO_MOD) mod tidy' and commit generated changes."; git diff; exit 1)

.PHONY: docs
docs: ## Serve documentation and open in the browser.
	sleep 5 && xdg-open http://localhost:6060/pkg/$$(sed -n 's/^module //p' go.mod)/ &
	$(GO_RUN) golang.org/x/tools/cmd/godoc@v0.1.7 -http=:6060 -v

.PHONY: spec
spec: test ## Print human-readable specification from unit tests.
	$(GO_TEST) -run $(GO_TESTS) $(GO_PACKAGES) -json | $(GO_RUN) github.com/invidian/go-test-to-spec@f3e247d

.PHONY: test-cover-browse
test-cover-browse: test-cover ## Run unit tests and browse coverage report.
	$(GO_CMD) tool cover -html=$(COVERPROFILE)

.PHONY: update-linters
update-linters: ## Update linters configuration.
	# Remove all enabled linters.
	sed -i '/^  enable:/q0' .golangci.yml
	# Then add all possible linters to config.
	$(GOLANGCI_LINT) linters | grep -E '^\S+:' | cut -d: -f1 | sort | sed 's/^/    - /g' | grep -v -E "($$(grep '^  disable:' -A 100 .golangci.yml  | grep -E '    - \S+$$' | awk '{print $$2}' | tr \\n '|' | sed 's/|$$//g'))" >> .golangci.yml

.PHONY: test-update-linters
test-update-linters: test-working-tree-clean ## Check if linters configuration is up to date.
	make update-linters
	@test -z "$$(git status --porcelain)" || (echo "Linter configuration outdated. Run 'make update-linters' and commit generated changes to fix."; exit 1)

.PHONY: help
help: ## Prints help message.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
