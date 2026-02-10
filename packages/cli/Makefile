BINARY := ../../dist/agentx
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GOFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

SCHEMA_SRC := ../../packages/schema/manifest.schema.json
SCHEMA_DST := internal/manifest/schema/manifest.schema.json
BRANDING_SRC := ../../branding.yaml
BRANDING_DST := internal/branding/branding.yaml

.PHONY: build test test-integration clean sync-schema check-schema sync-branding

sync-schema:
	@cp $(SCHEMA_SRC) $(SCHEMA_DST)
	@echo "Schema synced: $(SCHEMA_SRC) -> $(SCHEMA_DST)"

check-schema:
	@diff -q $(SCHEMA_SRC) $(SCHEMA_DST) > /dev/null 2>&1 || \
		(echo "ERROR: Schema out of sync. Run 'make sync-schema' in packages/cli/" && exit 1)
	@echo "Schema in sync."

sync-branding:
	@cp $(BRANDING_SRC) $(BRANDING_DST)
	@echo "Branding synced: $(BRANDING_SRC) -> $(BRANDING_DST)"

build: sync-schema sync-branding
	@mkdir -p ../../dist
	go build $(GOFLAGS) -o $(BINARY) .

test:
	go test ./...

test-integration:
	go test -tags=integration -v -count=1 ./test/integration/...

clean:
	rm -f $(BINARY)
