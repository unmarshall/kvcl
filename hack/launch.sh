#!/usr/bin/env bash
set -eo pipefail

echoErr() { echo "$@" 1>&2; }
echo "Killing stale etcd/apiserver..."
for p in $(pgrep -f envtest); do kill -9 $p;done
echo "Launching etcd+apiserver+scheduler..."
set -o allexport && source launch.env && set +o allexport
go run cmd/main.go -audit-logs=true

