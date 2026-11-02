.DEFAULT_GOAL := help
.PHONY: help db-up db-down migrate-up run check psql

help:
	@echo "Каркас для нового сервиса"
	@echo "db-up       — запустить PostgreSQL"
	@echo "db-down     — остановить PostgreSQL, сохранив данные"
	@echo "migrate-up  — запустить БД и применить миграции"
	@echo "run         — запустить сервис"
	@echo "check       — go test и go vet"
	@echo "psql        — открыть SQL-консоль"

db-up:
	docker compose up -d --wait db

db-down:
	docker compose down

migrate-up: db-up
	docker compose run --rm migrate up

run:
	go run ./cmd

check:
	go test ./...
	go vet ./...

psql:
	docker compose exec db psql -U app -d example_service
