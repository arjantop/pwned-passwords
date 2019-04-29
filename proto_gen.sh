#!/usr/bin/env bash

set -euxo pipefail

ROOT=$(dirname $0)

GRPC_GATEWAY_VERSION=$(go list -m all | grep github.com/grpc-ecosystem/grpc-gateway | cut -d " " -f 2)

GOOGLE_API_PATH=${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@${GRPC_GATEWAY_VERSION}/third_party/googleapis/

cd ${ROOT}/pwnedpasswords

protoc -I. -I${GOOGLE_API_PATH} --go_out=plugins=grpc:. pwned_passwords.proto
protoc -I. -I${GOOGLE_API_PATH} --grpc-gateway_out=logtostderr=true:. pwned_passwords.proto
protoc -I. -I${GOOGLE_API_PATH} --swagger_out=logtostderr=true:. pwned_passwords.proto
