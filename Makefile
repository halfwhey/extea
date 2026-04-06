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
release: _require_new_version ## Bump VERSION + README + main.go, commit, tag, push (CI runs goreleaser) (NEW_VERSION=vX.Y.Z)
	@OLD_VERSION=$$(cat VERSION 2>/dev/null | tr -d '[:space:]'); \
	OLD_NUM=$${OLD_VERSION#v}; \
	NEW_NUM=$(NEW_VERSION:v%=%); \
	printf '%s\n' "$(NEW_VERSION)" > VERSION; \
	sed -i "s|gh release download $$OLD_VERSION |gh release download $(NEW_VERSION) |g" README.md; \
	sed -i "s|extea_$${OLD_NUM}_|extea_$${NEW_NUM}_|g" README.md; \
	sed -i "s|^var Version = .*|var Version = \"$${NEW_NUM}\"|" main.go
	git add VERSION README.md main.go
	git commit -m "$(NEW_VERSION)"
	git tag "$(NEW_VERSION)"
	git push origin master
	git push origin "$(NEW_VERSION)"
