include .env

.PHONY: run migrate-up migrate-down genrate-tables

run:
	go run ./cmd/*.go

migrate-up:
	migrate -path ./migrations -database "$(DB_DSN)" up

migrate-down:
	migrate -path ./migrations -database "$(DB_DSN)" down

genrate-tables:
	sqlc generate