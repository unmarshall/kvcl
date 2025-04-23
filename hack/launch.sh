#!/usr/bin/env bash
set -eo pipefail

echoErr() { echo "$@" 1>&2; }
echo "Killing stale etcd/apiserver..."
for p in $(pgrep -f envtest); do kill -9 $p;done
set -o allexport && source launch.env && set +o allexport
if [[ -f "./bin/kvcl" ]]; then
  echo "Launching kvcl - which embeds kube-scheduler and also launches independent etcd and kube-apiserver processes"
  ./bin/kvcl
  for p in $(pgrep -f envtest); do kill -9 $p;done
else
  echoErr "ERR: kvcl is not at './bin/kvcl' - please run './hack/setup.sh' before executing './hack/launch.sh'"
fi

