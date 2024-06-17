package util

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	schedulerappconfig "k8s.io/kubernetes/cmd/kube-scheduler/app/config"
	"k8s.io/kubernetes/pkg/scheduler"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/apis/config/scheme"
)

func CreateSchedulerAppConfig(restCfg *rest.Config, schedulerConfigPath string) (*schedulerappconfig.Config, error) {
	client, eventsClient, err := createSchedulerClients(restCfg)
	if err != nil {
		return nil, err
	}
	eventBroadcaster := events.NewEventBroadcasterAdapter(eventsClient)
	informerFactory := scheduler.NewInformerFactory(client, 0)
	dynClient := dynamic.NewForConfigOrDie(restCfg)
	dynamicInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynClient, 0, corev1.NamespaceAll, nil)
	schedulerConfig, err := loadSchedulerConfig(schedulerConfigPath)
	if err != nil {
		return nil, err
	}
	return &schedulerappconfig.Config{
		ComponentConfig:    *schedulerConfig,
		Client:             client,
		InformerFactory:    informerFactory,
		DynInformerFactory: dynamicInformerFactory,
		EventBroadcaster:   eventBroadcaster,
		KubeConfig:         restCfg,
	}, nil
}

func loadSchedulerConfig(configPath string) (*config.KubeSchedulerConfiguration, error) {
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kube scheduler config file: %s: %w", configPath, err)
	}
	obj, gvk, err := scheme.Codecs.UniversalDecoder().Decode(configBytes, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode kube scheduler config file: %s: %w", configPath, err)
	}
	if cfgObj, ok := obj.(*config.KubeSchedulerConfiguration); ok {
		cfgObj.TypeMeta.APIVersion = gvk.GroupVersion().String()
		return cfgObj, nil
	}
	return nil, fmt.Errorf("couldn't decode as KubeSchedulerConfiguration, got %s: ", gvk)
}

func createSchedulerClients(restCfg *rest.Config) (kubernetes.Interface, kubernetes.Interface, error) {
	client, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create scheduler client: %w", err)
	}
	eventClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create scheduler event client: %w", err)
	}
	return client, eventClient, nil
}
