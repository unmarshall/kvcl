package control

import (
	"context"
	"fmt"
	"github.com/unmarshall/kvcl/api"
	"github.com/unmarshall/kvcl/pkg/common"
	"github.com/unmarshall/kvcl/pkg/util"
	schedulingv1 "k8s.io/api/scheduling/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	schedulerappconfig "k8s.io/kubernetes/cmd/kube-scheduler/app/config"
	"k8s.io/kubernetes/pkg/scheduler"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// NewControlPlane creates a new control plane. None of the components of the
// control-plane are initialized and started. Call Start to initialize and start the control-plane.
func NewControlPlane(vClusterBinaryAssetsPath string, kubeConfigPath string) api.ControlPlane {
	return &controlPlane{
		binaryAssetsPath: vClusterBinaryAssetsPath,
		kubeConfigPath:   kubeConfigPath,
	}
}

type controlPlane struct {
	// binaryAssetsPath is the path to the kube-api-server and etcd binaries.
	binaryAssetsPath string
	// kubeConfigPath is the kube config path for the virtual cluster.
	kubeConfigPath string
	// restConfig is the rest config to connect to the in-memory kube-api-server.
	restConfig *rest.Config
	// client connects to the in-memory kube-api-server.
	client client.Client
	// testEnvironment starts kube-api-server and etcd processes in-memory.
	testEnvironment *envtest.Environment
	// scheduler is the Kubernetes scheduler run in-memory.
	scheduler    *scheduler.Scheduler
	nodeControl  api.NodeControl
	podControl   api.PodControl
	eventControl api.EventControl
}

func (c *controlPlane) Start(ctx context.Context) error {
	slog.Info("Starting in-memory kube-api-server and etcd...")
	vEnv, cfg, k8sClient, err := c.startKAPIAndEtcd()
	if err != nil {
		return err
	}
	kubeConfigBytes, err := util.CreateKubeconfigFileForRestConfig(cfg)
	if err != nil {
		return err
	}
	kubeConfigPath, err := util.WriteKubeConfig(c.kubeConfigPath, kubeConfigBytes)
	if err != nil {
		return err
	}
	slog.Info("Wrote Kubeconfig file to connect to virtual controlPlane", "path", kubeConfigPath)
	c.testEnvironment = vEnv
	c.restConfig = cfg
	c.client = k8sClient
	c.nodeControl = NewNodeControl(k8sClient)
	c.podControl = NewPodControl(k8sClient)
	c.eventControl = NewEventControl(k8sClient)
	slog.Info("Starting in-memory kube-scheduler...")
	return c.startScheduler(ctx, c.restConfig)
}

func (c *controlPlane) Stop() error {
	slog.Info("Stopping in-memory kube-api-server and etcd...")
	if err := c.testEnvironment.Stop(); err != nil {
		slog.Warn("failed to stop in-memory kube-api-server and etcd", "error", err)
	}
	// once the context passed to the scheduler gets cancelled, the scheduler will stop as well.
	// No need to stop the scheduler explicitly.
	return nil
}

func (c *controlPlane) FactoryReset(ctx context.Context) error {
	slog.Info("Removing all nodes...")
	if err := c.NodeControl().DeleteAllNodes(ctx); err != nil {
		return fmt.Errorf("failed to delete all nodes: %w", err)
	}
	slog.Info("Removing all pods...")
	if err := c.PodControl().DeleteAllPods(ctx, common.DefaultNamespace); err != nil {
		return fmt.Errorf("failed to delete all pods: %w", err)
	}
	slog.Info("Removing all events...")
	if err := c.EventControl().DeleteAllEvents(ctx, common.DefaultNamespace); err != nil {
		return fmt.Errorf("failed to delete all events: %w", err)
	}
	slog.Info("Removing all priority classes...")
	if err := c.client.DeleteAllOf(ctx, &schedulingv1.PriorityClass{}); err != nil {
		return fmt.Errorf("failed to delete all priority classes: %w", err)
	}
	slog.Info("Removing all CSINodes ...")
	if err := c.client.DeleteAllOf(ctx, &storagev1.CSINode{}); err != nil {
		return fmt.Errorf("failed to delete all CSINodes: %w", err)
	}
	slog.Info("In-memory controlPlane factory reset successfully")
	return nil
}

func (c *controlPlane) NodeControl() api.NodeControl {
	if c.client == nil {
		slog.Error("controlPlane not started, first start the control plane and then call NodeControl")
		panic("controlPlane not started")
	}
	return NewNodeControl(c.client)
}

func (c *controlPlane) PodControl() api.PodControl {
	if c.client == nil {
		slog.Error("controlPlane not started, first start the control plane and then call NodeControl")
		panic("controlPlane not started")
	}
	return NewPodControl(c.client)
}

func (c *controlPlane) EventControl() api.EventControl {
	if c.client == nil {
		slog.Error("controlPlane not started, first start the control plane and then call NodeControl")
		panic("controlPlane not started")
	}
	return NewEventControl(c.client)
}

func (c *controlPlane) Client() client.Client {
	return c.client
}

func (c *controlPlane) startKAPIAndEtcd() (vEnv *envtest.Environment, cfg *rest.Config, k8sClient client.Client, err error) {

	etcdconfig := envtest.Etcd{}
	slog.Info("Modifying etcd config")
	etcdconfig.Configure().Append("auto-compaction-mode", "revision").Append("auto-compaction-retention", "5").Append("quota-backend-bytes", "8589934592")
	cpConfig := envtest.ControlPlane{Etcd: &etcdconfig}

	vEnv = &envtest.Environment{
		Scheme:                   scheme.Scheme,
		Config:                   nil,
		BinaryAssetsDirectory:    c.binaryAssetsPath,
		AttachControlPlaneOutput: true,
		ControlPlane:             cpConfig,
	}

	cfg, err = vEnv.Start()
	if err != nil {
		err = fmt.Errorf("failed to start virtual controlPlane: %w", err)
		return
	}
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		err = fmt.Errorf("failed to create client for virtual controlPlane: %w", err)
		return
	}
	return
}

func (c *controlPlane) startScheduler(ctx context.Context, restConfig *rest.Config) error {
	slog.Info("creating in-memory kube-scheduler configuration...")
	sac, err := util.CreateSchedulerAppConfig(restConfig)
	if err != nil {
		return err
	}
	recorderFactory := func(name string) events.EventRecorder {
		return sac.EventBroadcaster.NewRecorder(name)
	}
	s, err := scheduler.New(ctx,
		sac.Client,
		sac.InformerFactory,
		sac.DynInformerFactory,
		recorderFactory,
		scheduler.WithComponentConfigVersion(sac.ComponentConfig.TypeMeta.APIVersion),
		scheduler.WithKubeConfig(sac.KubeConfig),
		scheduler.WithProfiles(sac.ComponentConfig.Profiles...),
		scheduler.WithPercentageOfNodesToScore(sac.ComponentConfig.PercentageOfNodesToScore),
	)
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}
	c.scheduler = s
	sac.EventBroadcaster.StartRecordingToSink(ctx.Done())
	startInformersAndWaitForSync(ctx, sac, s)
	go func() {
		defer sac.EventBroadcaster.Shutdown()
		s.Run(ctx)
	}()
	slog.Info("in-memory kube-scheduler started successfully")
	return nil
}

func startInformersAndWaitForSync(ctx context.Context, sac *schedulerappconfig.Config, s *scheduler.Scheduler) {
	slog.Info("starting kube-scheduler informers...")
	sac.InformerFactory.Start(ctx.Done())
	if sac.DynInformerFactory != nil {
		sac.DynInformerFactory.Start(ctx.Done())
	}
	slog.Info("waiting for kube-scheduler informers to sync...")
	sac.InformerFactory.WaitForCacheSync(ctx.Done())
	if sac.DynInformerFactory != nil {
		sac.DynInformerFactory.WaitForCacheSync(ctx.Done())
	}
	if err := s.WaitForHandlersSync(ctx); err != nil {
		slog.Error("waiting for kube-scheduler handlers to sync", "error", err)
	}
}
