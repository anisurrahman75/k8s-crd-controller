#!/bin/bash

set -x
GROUP_NAME="mycrd.k8s"
VERSION_NAME="v1alpha1"
SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
vendor/k8s.io/code-generator/generate-groups.sh all \
github.com/anisurrahman75/k8s-crd-controller/pkg/client \
github.com/anisurrahman75/k8s-crd-controller/pkg/apis \
$GROUP_NAME:$VERSION_NAME \
--output-base "${GOPATH}/src" \
--go-header-file "hack/boilerplate.go.txt"