.PHONY: test
test: gomod
	bash -c 'set -a; . .env; set +a; go test  ./... -v --race --tags=tests'

test-cov:  ## Запустить тесты с покрытием
	go test --tags=tests -coverpkg=./internal/... -coverprofile=coverage.txt ./...
	go tool cover -func coverage.txt
	rm coverage.txt

test-ci:
	go test --tags=tests -v --race -coverpkg=./internal/...  ./... -json > test-results.json
