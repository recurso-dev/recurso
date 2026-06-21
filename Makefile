.PHONY: build run test test-e2e clean docker-up docker-down

BINARY_NAME=main

build:
	go build -o $(BINARY_NAME) cmd/api/main.go

run:
	go run cmd/api/main.go

test:
	go test -v ./...

test-e2e:
	@chmod +x scripts/e2e_test.sh
	./scripts/e2e_test.sh

clean:
	go clean
	rm -f $(BINARY_NAME)

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docs:
	cd docs-site && npm start

website:
	cd website && npm run dev

