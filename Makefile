include .env
export

RUN_CMD := go run cmd/main.go

start:
	$(RUN_CMD)

start-prod:
	@set -a && . ./.prod.env && set +a && $(RUN_CMD)

deepseek:
	MODEL=deepseek/deepseek-v3.2 PROVIDER=avian/fp8 $(RUN_CMD)

minimax:
	MODEL=minimax/minimax-m2 PROVIDER=google-vertex INCLUDE_REASONING=1 $(RUN_CMD)

gemini:
	MODEL=google/gemini-2.5-flash $(RUN_CMD)

gpt-mini:
	MODEL=openai/gpt-5-mini $(RUN_CMD)

gpt:
	MODEL=openai/gpt-5 $(RUN_CMD)

sonnet:
	MODEL=anthropic/claude-sonnet-4.5 PROVIDER=anthropic $(RUN_CMD)

haiku:
	MODEL=anthropic/claude-haiku-4.5 PROVIDER=anthropic $(RUN_CMD)

mistral:
	MODEL=mistralai/ministral-14b-2512 $(RUN_CMD)

build:
	go build -o main cmd/main.go
