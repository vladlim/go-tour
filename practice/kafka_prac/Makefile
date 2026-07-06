.PHONY: up, down, run, stop

up:
	@cd $(CURDIR) && docker-compose up -d

down:
	@cd $(CURDIR) && docker-compose down

run:
	@cd $(CURDIR) && go run cmd/main.go --cfg_path configs/config.yaml

stop:
	@-lsof -ti:8080 | xargs kill -9 2>/dev/null
	@echo "Port 8080 is free"
