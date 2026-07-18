-include .env
export

CONFIG ?=
RUN_CMD := go run ./cmd --config "$(CONFIG)"

kill:
	@-lsof -ti:8081 | xargs kill -9 2>/dev/null || true

start: kill
	@$(RUN_CMD) serve

start-prod: kill
	@set -a && . ./.prod.env && set +a && $(RUN_CMD) serve

validate:
	@$(RUN_CMD) validate

list:
	@$(RUN_CMD) list

.PHONY: dev
dev:
	@watchexec -q -e go,md,yaml -r make start CONFIG="$(CONFIG)"

build:
	go build -o main ./cmd
