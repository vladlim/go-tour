.PHONY: up down reset logs event producer-db consumer-db test

up:
	docker compose up --build -d

down:
	docker compose down

reset:
	docker compose down -v

logs:
	docker compose logs -f producer consumer

event:
	curl -i -X POST http://localhost:8081/events -H 'Content-Type: application/json' -d '{"message":"hello outbox"}'

producer-db:
	docker compose exec producer-db psql -U producer -d producer

consumer-db:
	docker compose exec consumer-db psql -U consumer -d consumer

test:
	go test ./...
