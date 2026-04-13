include .env
export

RUN_CMD := go run cmd/main.go

kill:
	@-lsof -ti:8081 | xargs kill -9 2>/dev/null || true

start: kill
	@$(RUN_CMD)

start-prod:
	@set -a && . ./.prod.env && set +a && $(RUN_CMD)

.PHONY: dev
dev:
	@watchexec -q -e go,md,json -r make start

prompt:
	@go run ./cmd/generate $(ARGS)

build:
	go build -o main cmd/main.go
