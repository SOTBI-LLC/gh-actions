.PHONY: gomod
gomod:
	@go mod tidy

run: gomod fullfmt ## Запустить сервис
	bash -c 'set -a; . .env; set +a; go run ./cmd/releasebot'

check: gomod fullfmt lint test
