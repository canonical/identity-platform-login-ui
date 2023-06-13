GO111MODULE?=on
CGO_ENABLED?=0
GOOS?=linux
GO?=go
MAIN_DIR?=/var/app

.EXPORT_ALL_VARIABLES:


mocks: vendor
	$(GO) install github.com/golang/mock/mockgen@v1.6.0
	# generate gomocks
	$(GO) generate ./...
.PHONY: mocks

test: mocks vet
	$(GO) test ./... -cover -coverprofile coverage_source.out
	# this will be cached, just needed to the test.json
	$(GO) test ./... -cover -coverprofile coverage_source.out -json > test_source.json
	cat coverage_source.out | grep -v "mock_*" | tee coverage.out
	cat test_source.json | grep -v "mock_*" | tee test.json
.PHONY: test

vet:
	$(GO) vet ./...
.PHONY: vet

vendor:
	$(GO) mod vendor
.PHONY: vendor

