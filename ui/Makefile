NPM?=npm


build: install
	$(NPM) run build --frozen=lockfile
.PHONY=build

install:
	$(NPM) install
.PHONY=install

test:
	$(NPM) ci 
.PHONY=test
