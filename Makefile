VERSION := $(shell cat VERSION 2>/dev/null | tr -d '[:space:]')

.PHONY: _require_new_version
_require_new_version:
	@[ -n "$(NEW_VERSION)" ] || (printf 'error: NEW_VERSION is not set. Usage: make release NEW_VERSION=vX.Y.Z\n' >&2; exit 1)

.PHONY: build
build: ## Build the extea binary
	CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=$(VERSION)" -o extea .

.PHONY: run
run: build ## Build and run extea (pass ARGS="..." for arguments)
	./extea $(ARGS)

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: release
release: _require_new_version ## Commit VERSION, tag, push, run goreleaser (NEW_VERSION=vX.Y.Z)
	printf '%s\n' "$(NEW_VERSION)" > VERSION
	git add VERSION
	git commit -m "$(NEW_VERSION)"
	git tag "$(NEW_VERSION)"
	git push origin master
	git push origin "$(NEW_VERSION)"
	goreleaser release --clean
