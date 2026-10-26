.PHONY: up down logs ps demo clean

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f core feed kafka

ps:
	docker compose ps

demo:
	curl -i -X POST 'http://localhost:8080/v1/posts' -H 'Content-Type: application/json' -H 'Idempotency-Key: demo-1' -d '{"title":"Hello outbox","body":"at-least-once delivery"}'

clean:
	docker compose down -v --remove-orphans
