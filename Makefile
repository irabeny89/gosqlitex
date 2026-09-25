.PHONY: test test-race test-bench

test:
	@go test -count=1 ./...

test-race:
	@go test -race -v ./...

test-bench:
	@go test -bench=. -benchmem ./...