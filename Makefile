.DEFAULT_GOAL := custom-verify

custom-verify: mod-verify go-mod-check

mod-verify: mod-verify-local
mod-verify-local:
	GO111MODULE=on go mod verify

go-mod-check: go-mod-check-local
go-mod-check-local:
	@echo make go-mod-check
	go mod tidy
	@if [ -n "$$(git status -s go.*)" ]; then \
		echo -e "${RED}✗ go mod tidy modified go.mod or go.sum files${NC}"; \
		git status -s go.*; \
		exit 1; \
	fi;

.PHONY: test
test:
	GOFIPS140=v1.0.0 go test ./...
