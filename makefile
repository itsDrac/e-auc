include .env
export

.PHONY: run migrate-up migrate-down generate-tables setup-dev docs

run:
	go run ./cmd/*.go

migrate-up:
	migrate -path ./migrations -database "$(DB_DSN)" up

migrate-down:
	migrate -path ./migrations -database "$(DB_DSN)" down

generate-tables:
	sqlc generate

docs:
	swag fmt
	swag init -g ./cmd/main.go -o ./docs

setup-dev:
	docker run -d --name postgres-dev -e POSTGRES_PASSWORD=password -e POSTGRES_DB=e-auc -p 5432:5432 postgres:alpine
	docker run -d --name minio-dev -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin -p 9000:9000 -p 9001:9001 minio/minio server /data --console-address ":9001"
	docker run -d --name redis-dev -p $(REDIS_PORT):6379 redis:latest redis-server --requirepass $(REDIS_PASSWORD)
