include .env
export

start:
	@echo "Start with one of: make deepseek, make minimax, make gemini, make gpt-mini, make gpt, make sonnet"

deepseek:
	MODEL=deepseek/deepseek-v3.2 provider=avian/fp8 go run cmd/main.go

minimax:
	MODEL=minimax/minimax-m2 PROVIDER=google-vertex INCLUDE_REASONING=1 go run cmd/main.go

gemini:
	MODEL=google/gemini-2.5-flash go run cmd/main.go

gpt-mini:
	MODEL=openai/gpt-5-mini go run cmd/main.go

gpt:
	MODEL=openai/gpt-5 go run cmd/main.go

sonnet:
	MODEL=anthropic/claude-sonnet-4.5 PROVIDER=anthropic go run cmd/main.go

build:
	go build -o main cmd/main.go
