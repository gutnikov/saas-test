# saas-test: one Go binary that serves the API and the built React frontend.

VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: ci web build test vet fmt-check typecheck dev-server dev-web clean

## ci: what CI runs: the frontend build, formatting, vet, tests and the binaries
ci: web fmt-check vet test build

## web: install the frontend's locked dependencies and build it into web/dist
web:
	cd web && npm ci && npm run build

## build: the server and the traffic generator into bin/, stamped with the git version
build: web/dist/index.html
	go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server
	go build -o bin/traffic ./cmd/traffic

test: web/dist/index.html
	go test ./...

vet: web/dist/index.html
	go vet ./...

fmt-check:
	@files=$$(gofmt -l cmd internal web/embed.go); \
	if [ -n "$$files" ]; then echo "not gofmt-ed:"; echo "$$files"; exit 1; fi

typecheck:
	cd web && npm run typecheck

## dev-server: the server on :8080 with ./saas-test.db
dev-server: web/dist/index.html
	go run -ldflags "$(LDFLAGS)" ./cmd/server

## dev-web: the Vite dev server on :5173; it proxies /api and /healthz to :8080
dev-web:
	cd web && npm run dev

# The server embeds web/dist. Until the frontend is built, a placeholder page stands in for it.
web/dist/index.html:
	mkdir -p web/dist
	printf '<!doctype html><title>saas-test</title><p>The frontend is not built: run make web.</p>\n' > $@

clean:
	rm -rf bin web/dist
