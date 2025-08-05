dev:			 ## Run the api in dev mode with hot reload
	go tool gow run ./cmd/api/main.go

lint:        ## Run linters and fix code, when possible (golangci-lint)
	go tool golangci-lint run --show-stats --fix

format: ##@ format lines of code to respect 100 characters
	go tool golines --no-reformat-tags -m 100 -w ./


crd:
	go tool controller-gen \
        	crd:crdVersions=v1 \
        	paths=./k8s/api/... \
        	output:crd:dir=./config/crd

generate:
	hack/generate-code.sh

.PHONY: crd generate