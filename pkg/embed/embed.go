package embed

import (
	_ "embed"
	"fmt"
	"k8s.io/client-go/kubernetes/scheme"
	schedulerconfig "k8s.io/kubernetes/pkg/scheduler/apis/config"
	schedulerconfigscheme "k8s.io/kubernetes/pkg/scheduler/apis/config/scheme"
)

var (
	//go:embed scheduler-config.yaml
	schedulerConfig string
)

// GetSchedulerConfig returns the embedded scheduler configuration.
func GetSchedulerConfig() (*schedulerconfig.KubeSchedulerConfiguration, error) {
	schedulerconfigscheme.AddToScheme(scheme.Scheme)
	obj, gvk, err := scheme.Codecs.UniversalDecoder().Decode([]byte(schedulerConfig), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode kube scheduler config %w", err)
	}
	if cfgObj, ok := obj.(*schedulerconfig.KubeSchedulerConfiguration); ok {
		cfgObj.TypeMeta.APIVersion = gvk.GroupVersion().String()
		return cfgObj, nil
	}
	return nil, fmt.Errorf("couldn't decode as KubeSchedulerConfiguration, got %s: ", gvk)
}
