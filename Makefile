export STORAGE_PORT=8080
export REST_PORT=8080
export REST_SERVICE_URL="rest-service:${REST_PORT}"

.PHONY: build run

build:
	docker compose -f ci-cd/build/compose.yml build #rest-service storage-service

run: build
	docker compose -f ci-cd/run/compose.yml up
