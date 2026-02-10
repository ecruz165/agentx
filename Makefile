.PHONY: build test clean install

# Orchestrate via pnpm (respects dependency order across packages)
build:
	pnpm -r run build

test:
	pnpm -r run test

clean:
	pnpm -r run clean

# Install all workspace dependencies
install:
	pnpm install
