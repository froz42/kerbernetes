#!/usr/bin/env bash

# Modified from https://github.com/kubernetes-sigs/kueue/blob/065451d907fa27a0647436505b3cac38718ef136/hack/update-codegen.sh
# Apache-2.0, Copyright 2023 The Kubernetes Authors

set -o errexit
set -o nounset
set -o pipefail

GO_CMD=${1:-go}
PKG_ROOT=$(realpath "$(dirname ${BASH_SOURCE[0]})/..")
CODEGEN_PKG=$($GO_CMD list -m -f "{{.Dir}}" k8s.io/code-generator)

cd $PKG_ROOT

source "${CODEGEN_PKG}/kube_codegen.sh"

# TODO: remove the workaround when the issue is solved in code-generator
# (https://github.com/kubernetes/code-generator/issues/165).
# kube_codegen.sh expects this layout:
# .
# └── github.com
#     └── froz42
#         └── kerbernetes
# We can use soft links in order to fake this layout, such that
ln -s . github.com
ln -s .. froz42
trap "rm github.com && rm froz42" EXIT

kube::codegen::gen_helpers \
  github.com/froz42/kerbernetes/k8s/api \
  --boilerplate /dev/null

kube::codegen::gen_client \
  github.com/froz42/kerbernetes/k8s/api \
  --output-dir github.com/froz42/kerbernetes/k8s/generated \
  --output-pkg github.com/froz42/kerbernetes/k8s/generated \
  --boilerplate /dev/null \
  --with-watch \
  --with-applyconfig

# clean up temporary libraries added in go.mod by code-generator
"${GO_CMD}" mod tidy