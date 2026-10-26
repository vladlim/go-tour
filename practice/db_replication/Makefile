.PHONY: up down reset logs test

up:
	docker compose up -d --wait

down:
	docker compose down

reset:
	docker compose down -v

logs:
	docker compose logs -f

test:
	go test ./...

