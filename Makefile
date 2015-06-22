.PHONY: all fmt vet lint test travis

all: deps fmt test

GIT_VERSION := $(shell git describe --abbrev=8 --dirty --always)
GO_LDFLAGS=-ldflags "-X `go list ./version`.Version $(GIT_VERSION)"

bridge: deps
	go build -o bridge $(GO_LDFLAGS) ./cmd/bridge_main.go


deps:
	go get -t ./cocaine12/...


fmt:
	@echo "+ $@"
	@test -z "$$(gofmt -s -l ./cocaine12/ | grep -v Godeps/_workspace/src/ | tee /dev/stderr)" || \
		echo "+ please format Go code with 'gofmt -s'"

lint:
	@echo "+ $@"
	@test -z "$$(golint ./cocaine12/... | grep -v Godeps/_workspace/src/ | grep -v cocaine12/old | tee /dev/stderr)"


vet:
	@echo "+ $@"
	@go vet ./cocaine12/...


test:
	@echo "+ $@"
	@go test -v -cover -race github.com/cocaine/cocaine-framework-go/cocaine12


