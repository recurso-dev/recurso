.PHONY: build run test test-e2e test-verify clean docker-up docker-down lint docker-build k8s-deploy k8s-status

BINARY_NAME=main
IMAGE_NAME=ghcr.io/recur-so/recurso
IMAGE_TAG?=latest

build:
	go build -o $(BINARY_NAME) cmd/api/main.go

run:
	go run cmd/api/main.go

test:
	go test -v ./...

test-e2e:
	@chmod +x scripts/e2e_test.sh
	./scripts/e2e_test.sh

# Phase verification scripts; require the dev stack (make docker-up) with
# APP_ENV=development and ALLOW_DEV_BYPASS=true.
test-verify:
	@chmod +x scripts/verify/*.sh
	@for s in scripts/verify/verify_p*.sh; do echo "== $$s =="; $$s || exit 1; done

clean:
	go clean
	rm -f $(BINARY_NAME)

lint:
	golangci-lint run --timeout=5m ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

k8s-deploy:
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/secret.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml
	kubectl apply -f k8s/ingress.yaml

k8s-status:
	kubectl -n recurso get pods
	kubectl -n recurso get svc
	kubectl -n recurso get ingress

docs:
	cd docs-site && npm start

website:
	cd website && npm run dev
