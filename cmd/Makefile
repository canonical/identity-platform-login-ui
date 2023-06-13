GO111MODULE?=on
CGO_ENABLED?=0
GOOS?=linux
GO_BIN?=app
GO?=go
GOFLAGS?=-ldflags=-w -ldflags=-s -a -buildvcs


.EXPORT_ALL_VARIABLES:

build:
	$(GO) build -o $(GO_BIN) ./

.PHONY=build