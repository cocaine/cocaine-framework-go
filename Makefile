.PHONY: all fmt vet lint test travis

all: deps fmt test


deps:
	go get -t ./...


fmt:
	@echo "+ $@"
	@test -z "$$(gofmt -s -l . | grep -v Godeps/_workspace/src/ | tee /dev/stderr)" || \
		echo "+ please format Go code with 'gofmt -s'"

lint:
	@echo "+ $@"
	@test -z "$$(golint ./... | grep -v Godeps/_workspace/src/ | tee /dev/stderr)"


vet:
	@echo "+ $@"
	@go vet ./...


test:
	@echo "+ $@"
	@go test -v -cover -race github.com/cocaine/cocaine-framework-go/cocaine


