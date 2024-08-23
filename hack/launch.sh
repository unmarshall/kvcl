#!/usr/bin/env bash
set -eo pipefail

echoErr() { echo "$@" 1>&2; }
echo "Killing stale etcd/apiserver..."
for p in $(pgrep -f envtest); do kill -9 $p;done
echo "Launching etcd+apiserver+scheduler..."
go run cmd/main.go

