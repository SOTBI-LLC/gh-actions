.PHONY: gomod
gomod:
	@go mod tidy

run: gomod fullfmt ## Запустить сервис
	bash -c 'set -a; . .env; set +a; go run ./cmd/releasebot/main.go'

check: gomod fullfmt lint test
