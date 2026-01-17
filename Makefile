include .env
export

RUN_CMD := go run cmd/main.go

start:
	@$(RUN_CMD)

start-prod:
	@set -a && . ./.prod.env && set +a && $(RUN_CMD)

.PHONY: dev
dev:
	@watchexec -q -e go -r make start

deepseek:
	MODEL=deepseek/deepseek-v3.2 PROVIDER=avian/fp8 $(RUN_CMD)

minimax:
	MODEL=minimax/minimax-m2 PROVIDER=google-vertex INCLUDE_REASONING=1 $(RUN_CMD)

gemini:
	MODEL=google/gemini-2.5-flash $(RUN_CMD)

gpt-mini:
	MODEL=gpt-5-mini BASE_URL=https://api.openai.com/v1 GPT_VERBOSITY=medium REASONING_EFFORT= $(RUN_CMD)

gpt:
	MODEL=gpt-5.1 BASE_URL=https://api.openai.com/v1 INCLUDE_REASONING= $(RUN_CMD)

gpt-4:
	MODEL=gpt-4.1 BASE_URL=https://api.openai.com/v1 $(RUN_CMD)

sonnet:
	MODEL=anthropic/claude-sonnet-4.5 PROVIDER=anthropic $(RUN_CMD)

haiku:
	MODEL=anthropic/claude-haiku-4.5 PROVIDER=anthropic $(RUN_CMD)

mistral:
	MODEL=mistralai/ministral-14b-2512 $(RUN_CMD)

build:
	go build -o main cmd/main.go
