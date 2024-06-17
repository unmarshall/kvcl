package util

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func CreateKubeconfigFileForRestConfig(restConfig *rest.Config) ([]byte, error) {
	clusters := make(map[string]*clientcmdapi.Cluster)
	clusters["default-controlPlane"] = &clientcmdapi.Cluster{
		Server:                   restConfig.Host,
		CertificateAuthorityData: restConfig.CAData,
	}
	contexts := make(map[string]*clientcmdapi.Context)
	contexts["default-context"] = &clientcmdapi.Context{
		Cluster:  "default-controlPlane",
		AuthInfo: "default-user",
	}
	authInfos := make(map[string]*clientcmdapi.AuthInfo)
	authInfos["default-user"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: restConfig.CertData,
		ClientKeyData:         restConfig.KeyData,
	}
	clientConfig := clientcmdapi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		Contexts:       contexts,
		CurrentContext: "default-context",
		AuthInfos:      authInfos,
	}
	kubeConfigBytes, err := clientcmd.Write(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig for virtual enviroment: %w", err)
	}
	return kubeConfigBytes, nil
}

func WriteKubeConfig(kubeConfigBytes []byte) (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	kubeConfigDir := filepath.Join(currentDir, "tmp")
	if _, err = os.Stat(kubeConfigDir); os.IsNotExist(err) {
		err = os.Mkdir(kubeConfigDir, 0755)
		if err != nil {
			return "", err
		}
	}
	kubeConfigPath := filepath.Join(kubeConfigDir, "kubeconfig.yaml")
	if err = os.WriteFile(kubeConfigPath, kubeConfigBytes, 0644); err != nil {
		return "", err
	}
	slog.Info("KubeConfig to connect to Virtual Custer written to: " + kubeConfigPath)
	return kubeConfigPath, nil
}
