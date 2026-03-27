docker-build:
	@docker compose -f docker-compose.yml build

docker-up:
	@docker compose -f docker-compose.yml up --build

docker-local-up:
	@docker compose -f docker-compose.local.yml up --build

docker-down:
	@docker compose -f docker-compose.yml down

docker-local-down:
	@docker compose -f docker-compose.local.yml down

monthly-settlement:
	@docker compose -f docker-compose.yml run --rm monthly_settlement

migrate-%:
	go run ./cmd/migration/main.go schema apply -r ./database/migration/${@:migrate-%=%} -p 5432 --dbname postgres -u myadmin -P ${POSTGRES_PASSWORD}

.PHONY: migrate
migrate:
	make migrate-uasl_reservation

.PHONY: seed
seed:
	PGPASSWORD=${POSTGRES_PASSWORD} psql -h localhost -p 5432 -U myadmin -d postgres -f ./database/migration/seed/seed_data.sql
