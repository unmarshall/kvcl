package util

import (
	"fmt"
	"github.com/unmarshall/kvcl/pkg/embed"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	schedulerappconfig "k8s.io/kubernetes/cmd/kube-scheduler/app/config"
	"k8s.io/kubernetes/pkg/scheduler"
)

func CreateSchedulerAppConfig(kubeConfigPath string, restCfg *rest.Config) (*schedulerappconfig.Config, error) {
	client, eventsClient, err := createSchedulerClients(restCfg)
	if err != nil {
		return nil, err
	}
	eventBroadcaster := events.NewEventBroadcasterAdapter(eventsClient)
	informerFactory := scheduler.NewInformerFactory(client, 0)
	dynClient := dynamic.NewForConfigOrDie(restCfg)
	dynamicInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynClient, 0, corev1.NamespaceAll, nil)
	schedulerConfig, err := embed.GetSchedulerConfig()
	schedulerConfig.ClientConnection.Kubeconfig = kubeConfigPath
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
