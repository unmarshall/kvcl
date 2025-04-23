# k8s-virtual-cluster
Virtual cluster creates an in-memory k8s control-plane which comprises of the following components:
* Kube API Server
* Single node Etcd
* Kube Scheduler


## Usage

### Prerequisites
* Ensure that you have installed `envtest`. You can just invoke `./hack/setup.sh` to install `envtest`. This will also create
a `launch.env` in the project root directory.
* Execute `set -o allexport && source launch.env && set +o allexport` this will set key-value pairs populated in `launch.env` as environment variables.

### Running the virtual cluster

Start the virtual cluster by running the following command:
```bash
./hack/launch.sh [flags]
OR 
go run cmd/main.go [flags]
```
**Flags**:
* `--target-kvcl-kubeconfig` : Path where the kubeconfig to connect to the virtual cluster will be written. Default value is `/tmp/kvcl.yaml`
