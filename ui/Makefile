NPM?=npm
NPX?=npx


build: install
	$(NPM) run build --frozen=lockfile
.PHONY=build

install:
	$(NPM) install
.PHONY=install

test:
	$(NPM) ci 
.PHONY=test

test-e2e:
	$(NPX) playwright test
.PHONY=test-e2e

test-e2e-debug:
	$(NPX) playwright test --ui
.PHONY=test-e2e-debug
