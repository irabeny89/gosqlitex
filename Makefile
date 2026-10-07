.PHONY: test test-race test-bench

test:
	@go test ./...

test-race:
	@go test -race -v ./...

test-bench:
	@go test -bench=. -benchmem ./...