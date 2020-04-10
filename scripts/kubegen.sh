#!/usr/bin/env sh

set -o errexit
set -o nounset

IMAGE=quay.io/slok/kube-code-generator:v1.18.0
ROOT_DIRECTORY=$(dirname "$(readlink -f "$0")")/../
PROJECT_PACKAGE="github.com/slok/bifrost"

echo "Generating Kubernetes CRD clients..."
docker run -it --rm \
	-v ${ROOT_DIRECTORY}:/go/src/${PROJECT_PACKAGE} \
	-e PROJECT_PACKAGE=${PROJECT_PACKAGE} \
	-e CLIENT_GENERATOR_OUT=${PROJECT_PACKAGE}/pkg/kubernetes/gen \
	-e APIS_ROOT=${PROJECT_PACKAGE}/pkg/apis \
	-e GROUPS_VERSION="auth:v1" \
	-e GENERATION_TARGETS="deepcopy,client" \
	${IMAGE}

echo "Generating Kubernetes CRD manifests..."
docker run -it --rm \
	-v ${ROOT_DIRECTORY}:/src \
	-e GO_PROJECT_ROOT=/src \
	-e CRD_TYPES_PATH=/src/pkg/apis \
	-e CRD_OUT_PATH=/src/manifests \
	${IMAGE} update-crd.sh