.PHONY: test test-prepare test-unit test-int test-e2e

export PATH := $(CURDIR)/scripts:$(PATH)
export GOPROXY := off
export GOSUMDB := off

test:
	$(MAKE) test-prepare
	$(MAKE) test-unit
	$(MAKE) test-int
	$(MAKE) test-e2e

test-prepare:
	find cmd internal scripts test -type f -name '*.go' -print0 | xargs -0 gofmt -w
	go vet ./...
	golangci-lint run ./...
	go build ./...
	go run scripts/check-docs.go

test-unit:
	ginkgo --race --randomize-all --fail-on-empty --fail-on-pending ./internal/domain ./internal/options ./internal/render ./internal/provider/envfilter

test-int:
	ginkgo --race --randomize-all --fail-on-empty --fail-on-pending ./internal/app ./internal/process ./internal/provider/claude ./internal/provider/codex ./internal/provider/grok ./internal/rpc ./cmd/gaebokchi ./scripts

test-e2e:
	ginkgo --race --randomize-all --fail-on-empty --fail-on-pending ./test/integration
