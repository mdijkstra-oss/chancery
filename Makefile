# .env.local and not .env: .env holds the compose stack's provider keys, which
# belong to the dragoman container, and sourcing it here would put them in this
# process's environment for no reason.
-include .env.local
export

CONFIG ?= ./config
RUN_CMD := go run ./cmd/chancery --config "$(CONFIG)"

kill:
	@-lsof -ti:8081 | xargs kill -9 2>/dev/null || true

start: kill
	@$(RUN_CMD) serve

start-prod: kill
	@set -a && . ./.prod.env.local && set +a && $(RUN_CMD) serve

validate:
	@$(RUN_CMD) validate

list:
	@$(RUN_CMD) list

.PHONY: dev
dev:
	@watchexec -q -e go,md,yaml -r make start CONFIG="$(CONFIG)"

build:
	@mkdir -p bin
	go build -o bin/chancery ./cmd/chancery

.PHONY: test test-race vet fmt fmt-check lint cover
test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@files=$$(gofmt -l .); if [ -n "$$files" ]; then echo "$$files"; exit 1; fi

lint:
	golangci-lint run

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
