package hack

// Keep a reference to code-generator so it's not removed by go mod tidy
import (
	_ "k8s.io/code-generator"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
