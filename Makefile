.PHONY: build run seed demo test test-e2e test-verify clean docker-up docker-down lint docker-build k8s-deploy k8s-status

BINARY_NAME=main
IMAGE_NAME=ghcr.io/swapnull-in/recur-so
IMAGE_TAG?=latest
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS=-X main.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) cmd/api/main.go

run:
	go run cmd/api/main.go

# DESTRUCTIVE: wipes all data in the target database, then loads demo data
# (tenant "Acme SaaS Corp", API key sk_test_12345, plans/customers/invoices).
seed:
	@echo "WARNING: this wipes ALL data in $${DATABASE_URL:-the local dev database} and loads demo data."
	go run cmd/seed/main.go

# One-command demo: builds and starts the full stack (API + dashboard),
# waits for the API to become healthy, then loads demo data.
demo:
	docker-compose up -d --build
	@echo "Waiting for API at http://localhost:8080/health ..."
	@for i in $$(seq 1 60); do \
		if curl -sf http://localhost:8080/health > /dev/null 2>&1; then \
			echo "API is up."; break; \
		fi; \
		if [ $$i -eq 60 ]; then \
			echo "ERROR: API did not become healthy within 60s." >&2; exit 1; \
		fi; \
		sleep 1; \
	done
	DATABASE_URL="postgres://user:password@localhost:5432/recurso?sslmode=disable" go run cmd/seed/main.go
	@echo ""
	@echo "=================================================================="
	@echo "  Recurso demo is ready!"
	@echo ""
	@echo "  Dashboard:  http://localhost:5173  (log in with API key sk_test_12345)"
	@echo "  API:        http://localhost:8080"
	@echo "  Emails:     http://localhost:8025  (Mailhog)"
	@echo "=================================================================="

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
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE_NAME):$(IMAGE_TAG) .

k8s-deploy:
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/rbac.yaml
	kubectl apply -f k8s/networkpolicy.yaml
	kubectl apply -f k8s/configmap.yaml
	kubectl apply -f k8s/secret.yaml
	kubectl apply -f k8s/deployment.yaml
	kubectl apply -f k8s/service.yaml
	kubectl apply -f k8s/ingress.yaml

k8s-status:
	kubectl -n recurso get pods
	kubectl -n recurso get svc
	kubectl -n recurso get ingress

website:
	cd website && npm run dev
