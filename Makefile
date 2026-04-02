.PHONY: help build run test lint clean docker-build docker-up docker-down docker-logs \
		k8s-apply k8s-delete k8s-logs k8s-status k8s-restart k8s-scale \
		deploy-staging deploy-production

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
IMAGE_NAME := bm-photo
K8S_DIR := ./kubernetes
NAMESPACE := bm-photo

# Default target
help: ## Show this help message
	@echo "BM Photo Service - Deployment Commands"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Build Targets:"
	@echo "  build          Build the Go binary"
	@echo "  run            Run the application locally"
	@echo "  test           Run tests"
	@echo "  lint           Run linters"
	@echo "  clean          Clean build artifacts"
	@echo ""
	@echo "Docker Targets:"
	@echo "  docker-build   Build Docker image"
	@echo "  docker-up      Start Docker Compose services"
	@echo "  docker-down    Stop Docker Compose services"
	@echo "  docker-logs    Show Docker Compose logs"
	@echo "  docker-clean   Remove Docker Compose containers and volumes"
	@echo ""
	@echo "Kubernetes Targets:"
	@echo "  k8s-apply      Apply Kubernetes manifests"
	@echo "  k8s-delete     Delete Kubernetes resources"
	@echo "  k8s-logs       Show pod logs"
	@echo "  k8s-status     Show deployment status"
	@echo "  k8s-restart   Restart deployment"
	@echo "  k8s-scale      Scale deployment"
	@echo ""
	@echo "Deployment Targets:"
	@echo "  deploy-staging     Deploy to staging"
	@echo "  deploy-production  Deploy to production"
	@echo ""

# Build the Go binary
build: ## Build the Go binary
	@echo "Building bm-photo..."
	CGO_ENABLED=0 go build -ldflags " \
		-X main.version=$(VERSION) \
		-X main.buildTime=$(BUILD_TIME) \
	" -o bin/server ./cmd/server
	@echo "Build complete: bin/server"

# Run the application locally
run: build ## Run the application locally
	@echo "Starting bm-photo server..."
	./bin/server

# Run tests
test: ## Run tests
	@echo "Running tests..."
	go test -v -race -cover ./...

# Run tests with coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linters
lint: ## Run linters
	@echo "Running linters..."
	golangci-lint run ./...
	gofmt -s -w .
	@echo "Lint complete"

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image: $(IMAGE_NAME):$(VERSION)"
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest \
		.
	@echo "Docker image built: $(IMAGE_NAME):$(VERSION)"

docker-build-no-cache: ## Build Docker image without cache
	@echo "Building Docker image without cache: $(IMAGE_NAME):$(VERSION)"
	docker build --no-cache \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest \
		.
	@echo "Docker image built: $(IMAGE_NAME):$(VERSION)"

docker-up: ## Start Docker Compose services
	@echo "Starting Docker Compose services..."
	docker compose up -d
	@echo "Services started. Run 'make docker-logs' to view logs."

docker-down: ## Stop Docker Compose services
	@echo "Stopping Docker Compose services..."
	docker compose down

docker-logs: ## Show Docker Compose logs
	docker compose logs -f

docker-clean: ## Remove Docker Compose containers and volumes
	@echo "Cleaning Docker Compose resources..."
	docker compose down -v --remove-orphans
	@echo "Docker Compose resources cleaned"

# Kubernetes targets
k8s-apply: ## Apply Kubernetes manifests
	@echo "Applying Kubernetes manifests in $(K8S_DIR)..."
	kubectl apply -f $(K8S_DIR)/00-namespace.yaml
	kubectl apply -f $(K8S_DIR)/01-configmap.yaml
	kubectl apply -f $(K8S_DIR)/02-secret.yaml
	kubectl apply -f $(K8S_DIR)/03-deployment.yaml
	kubectl apply -f $(K8S_DIR)/04-service-account.yaml
	kubectl apply -f $(K8S_DIR)/05-ingress.yaml
	kubectl apply -f $(K8S_DIR)/06-hpa.yaml
	@echo "Kubernetes manifests applied"

k8s-delete: ## Delete Kubernetes resources
	@echo "Deleting Kubernetes resources in $(NAMESPACE) namespace..."
	kubectl delete -f $(K8S_DIR)/06-hpa.yaml --ignore-not-found=true
	kubectl delete -f $(K8S_DIR)/05-ingress.yaml --ignore-not-found=true
	kubectl delete -f $(K8S_DIR)/04-service-account.yaml --ignore-not-found=true
	kubectl delete -f $(K8S_DIR)/03-deployment.yaml --ignore-not-found=true
	kubectl delete -f $(K8S_DIR)/02-secret.yaml --ignore-not-found=true
	kubectl delete -f $(K8S_DIR)/01-configmap.yaml --ignore-not-found=true
	kubectl delete -f $(K8S_DIR)/00-namespace.yaml --ignore-not-found=true
	@echo "Kubernetes resources deleted"

k8s-logs: ## Show pod logs
	@echo "Showing logs for bm-photo pods..."
	kubectl logs -n $(NAMESPACE) -l app.kubernetes.io/name=bm-photo -f --tail=100

k8s-status: ## Show deployment status
	@echo "Deployment status:"
	@kubectl get all -n $(NAMESPACE)
	@echo ""
	@echo "Pod status:"
	@kubectl get pods -n $(NAMESPACE) -o wide

k8s-restart: ## Restart deployment
	@echo "Restarting bm-photo deployment..."
	kubectl rollout restart deployment/bm-photo -n $(NAMESPACE)
	kubectl rollout status deployment/bm-photo -n $(NAMESPACE)
	@echo "Deployment restarted"

k8s-scale: ## Scale deployment (usage: make k8s-scale REPLICAS=5)
	@echo "Scaling bm-photo deployment to $(REPLICAS) replicas..."
	kubectl scale deployment/bm-photo -n $(NAMESPACE) --replicas=$(REPLICAS)

k8s-describe: ## Show deployment details
	@echo "Describing bm-photo deployment:"
	@kubectl describe deployment bm-photo -n $(NAMESPACE)
	@echo ""
	@echo "Describing pods:"
	@kubectl describe pods -n $(NAMESPACE) -l app.kubernetes.io/name=bm-photo

k8s-exec: ## Execute command in pod (usage: make k8s-exec CMD="ls -la")
	@POD=$$(kubectl get pods -n $(NAMESPACE) -l app.kubernetes.io/name=bm-photo -o jsonpath='{.items[0].metadata.name}'); \
	echo "Executing in pod: $$POD"; \
	kubectl exec -n $(NAMESPACE) -it $$POD -- $(CMD)

k8s-port-forward: ## Port forward to local (usage: make k8s-port-forward LOCAL_PORT=8080)
	@echo "Port forwarding to bm-photo service..."
	kubectl port-forward -n $(NAMESPACE) svc/bm-photo $(LOCAL_PORT):80

# Deployment targets
deploy-staging: docker-build ## Deploy to staging
	@echo "Deploying to staging..."
	kubectl config use-context staging
	docker tag $(IMAGE_NAME):$(VERSION) gcr.io/bm-photo-staging/$(IMAGE_NAME):$(VERSION)
	docker push gcr.io/bm-photo-staging/$(IMAGE_NAME):$(VERSION)
	kubectl set image deployment/bm-photo app=$(IMAGE_NAME):$(VERSION) -n $(NAMESPACE)
	kubectl rollout status deployment/bm-photo -n $(NAMESPACE)
	@echo "Deployed to staging"

deploy-production: docker-build ## Deploy to production
	@echo "Deploying to production..."
	@read -p "Are you sure you want to deploy to production? (yes/no) " ans; \
	if [ "$$ans" != "yes" ]; then \
		echo "Deployment cancelled."; \
		exit 1; \
	fi
	kubectl config use-context production
	docker tag $(IMAGE_NAME):$(VERSION) gcr.io/bm-photo-prod/$(IMAGE_NAME):$(VERSION)
	docker push gcr.io/bm-photo-prod/$(IMAGE_NAME):$(VERSION)
	kubectl set image deployment/bm-photo app=$(IMAGE_NAME):$(VERSION) -n $(NAMESPACE)
	kubectl rollout status deployment/bm-photo -n $(NAMESPACE)
	@echo "Deployed to production"
