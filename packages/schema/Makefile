.PHONY: build test clean

build:
	npm install --prefer-offline
	node scripts/validate-schemas.mjs

test:
	node --test test/*.test.mjs

clean:
	rm -rf node_modules
