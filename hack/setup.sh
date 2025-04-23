PROJECT_DIR="$(cd "$(dirname "${SCRIPT_DIR}")" &>/dev/null && pwd)"
LAUNCH_ENV_FILE="launch.env"
LAUNCH_ENV_PATH="$PROJECT_DIR/$LAUNCH_ENV_FILE"

function echoErr() {
    printf "%s\n" "$*" >&2
}

function setup_envtest() {
   local errorCode
   printf "Installing setup-envtest...\n"
   GOOS=$(go env GOOS)
   GOARCH=$(go env GOARCH)
   printf "GOOS=%s, GOARCH=%s\n" $GOOS $GOARCH
   go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
   envTestSetupCmd="setup-envtest --os $GOOS --arch $GOARCH use  -p path"
   printf "Executing: %s\n" "$envTestSetupCmd"
   binaryAssetsDir=$(eval "$envTestSetupCmd")
   errorCode="$?"
   if [[ "$errorCode" -gt 0 ]]; then
        echoErr "EC: $errorCode. Error in executing $envTestSetupCmd. Exiting!"
        exit 1
   fi

  printf "BINARY_ASSETS_DIR=\"%s\"" "$binaryAssetsDir"  > "$LAUNCH_ENV_PATH"
  printf "Wrote env to %s\n" "$LAUNCH_ENV_PATH"
}

setup_envtest "$@"
echo
echo "Building KVCL..."
[ -d bin ] || mkdir bin
go build -o bin/kvcl -v cmd/main.go
echo "NOTE: You can now run ./hack/launch.sh which will launch etcd process, kube-apiserver process and kvcl process that embeds the kube-scheduler"
