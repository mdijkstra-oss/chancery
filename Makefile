include .env
export

start:
	go run cmd/main.go

build:
	go build -o main cmd/main.go
