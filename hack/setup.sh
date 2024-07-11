PROJECT_DIR="$(cd "$(dirname "${SCRIPT_DIR}")" &>/dev/null && pwd)"
LAUNCH_ENV_FILE="launch.env"
LAUNCH_ENV_PATH="$PROJECT_DIR/$LAUNCH_ENV_FILE"
KUBE_SOURCE_DIR="$HOME/go/src/github.com/kubernetes/kubernetes"

function main() {
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

  if [[ ! -d $KUBE_SOURCE_DIR ]]; then
      echoErr "Please checkout k8s sources into $KUBE_SOURCE_DIR"
      exit 1
  fi

   if [[ ! -f "$binaryAssetsDir/kube-scheduler" ]]; then
     echo -e "No kube-scheduler binary in: $binaryAssetsDir"
     echo "Building kube-scheduler..."
     pushd "$KUBE_SOURCE_DIR" || exit 1
     go build -v -o /tmp/kube-scheduler cmd/kube-scheduler/scheduler.go
     chmod +w "$binaryAssetsDir"
     cp -v /tmp/kube-scheduler "$binaryAssetsDir"
     ls -al "$binaryAssetsDir/kube-scheduler"
     popd || exit 1
   fi

   if [[ ! -f "$binaryAssetsDir/kube-scheduler" ]]; then
     echo -e "No kube-scheduler binary in: $binaryAssetsDir"
     echo "Building kube-scheduler..."
     pushd "$KUBE_SOURCE_DIR" || exit 1
     go build -v -o /tmp/kube-scheduler cmd/kube-scheduler/scheduler.go
     chmod +w "$binaryAssetsDir"
     cp -v /tmp/kube-scheduler "$binaryAssetsDir"
     ls -al "$binaryAssetsDir/kube-scheduler"
     popd || exit 1
   fi

  printf "BINARY_ASSETS_DIR=\"%s\"" "$binaryAssetsDir"  > "$LAUNCH_ENV_PATH"
  printf "Wrote env to %s\n" "$LAUNCH_ENV_PATH"
  echo
  echo "NOTE: COPY & EXECUTE THIS->> set -o allexport && source launch.env && set +o allexport"
  echo "Then launch virtual cluster using go run main.go"
}
main "$@"