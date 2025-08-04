crd:
	go run sigs.k8s.io/controller-tools/cmd/controller-gen \
        	crd:crdVersions=v1 \
        	paths=./k8s/api/... \
        	output:crd:dir=./config/crd

generate:
	hack/generate-code.sh

.PHONY: crd generate