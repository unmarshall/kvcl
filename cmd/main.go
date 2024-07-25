package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/unmarshall/kvcl/api"
	"github.com/unmarshall/kvcl/pkg/control"
	"github.com/unmarshall/kvcl/pkg/util"
)

type config struct {
	binaryAssetsPath          string
	startScalingRecommender   bool
	targetClusterCAConfigPath string
	kubeConfigPath            string
}

const defaultKVCLKubeConfigPath = "/tmp/kvcl.yaml"

func main() {
	defer util.OnExit()
	var (
		vCluster api.ControlPlane
		err      error
	)
	ctx := setupSignalHandler()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg, err := parseCmdArgs()
	if err != nil {
		util.ExitAppWithError(1, fmt.Errorf("failed to parse cmd args :%w", err))
	}
	// kubeConfigPath is where the kubeconfig file will be written to for any consumer to access the virtual cluster.
	defer func() {
		logger.Info("shutting down virtual cluster...")
		if err = vCluster.Stop(); err != nil {
			logger.Error("failed to stop virtual cluster", "error", err)
		}
	}()
	logger.Info("starting virtual cluster", "config", cfg)
	vCluster, err = startVirtualCluster(ctx, cfg.binaryAssetsPath, cfg.kubeConfigPath)
	if err != nil {
		util.ExitAppWithError(1, fmt.Errorf("failed to start virtual cluster: %w", err))
	}
	<-ctx.Done()
}

func startVirtualCluster(ctx context.Context, binaryAssetsDir string, kubeConfigPath string) (api.ControlPlane, error) {
	vCluster := control.NewControlPlane(binaryAssetsDir, kubeConfigPath)
	if err := vCluster.Start(ctx); err != nil {
		slog.Error("failed to start virtual cluster", "error", err)
		return vCluster, err
	}
	slog.Info("virtual cluster started successfully")
	return vCluster, nil
}

func setupSignalHandler() context.Context {
	quit := make(chan os.Signal, 2)
	ctx, cancel := context.WithCancel(context.Background())
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		cancel()
		<-quit
		os.Exit(1)
	}()
	return ctx
}

func parseCmdArgs() (config, error) {
	cfg := config{}
	args := os.Args[1:]
	fs := flag.CommandLine
	fs.StringVar(&cfg.binaryAssetsPath, "binary-assets-dir", "", "Path to the binary assets for etcd and kube-apiserver")
	fs.StringVar(&cfg.kubeConfigPath, "kube-config-path", defaultKVCLKubeConfigPath, "Path where the kubeconfig file for the virtual cluster is written")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if err := cfg.resolveBinaryAssetsPath(); err != nil {
		return cfg, err
	}

	// ensure that targetClusterCAConfigPath is set when startScalingRecommender is set to true.
	if cfg.startScalingRecommender && len(strings.TrimSpace(cfg.targetClusterCAConfigPath)) == 0 {
		return cfg, fmt.Errorf("target-cluster-ca-config-path is required when start-scaling-recommender is set to true")
	}
	return cfg, nil
}

func (c *config) resolveBinaryAssetsPath() error {
	if c.binaryAssetsPath == "" {
		c.binaryAssetsPath = getBinaryAssetsPathFromEnv()
	}
	if c.binaryAssetsPath == "" {
		return fmt.Errorf("cannot find binary-assets-path")
	}
	return nil
}

func getBinaryAssetsPathFromEnv() string {
	return os.Getenv("BINARY_ASSETS_DIR")
}
