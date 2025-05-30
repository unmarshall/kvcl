package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
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
	auditLogs                 bool
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
		if vCluster != nil {
			if err = vCluster.Stop(); err != nil {
				logger.Error("failed to stop virtual cluster", "error", err)
			}
		}
	}()
	logger.Info("starting virtual cluster", "embed", cfg)
	vCluster, err = startVirtualCluster(ctx, cfg.binaryAssetsPath, cfg.kubeConfigPath, cfg.auditLogs)
	if err != nil {
		util.ExitAppWithError(1, fmt.Errorf("failed to start virtual cluster: %w", err))
	}
	<-ctx.Done()
}

func startVirtualCluster(ctx context.Context, binaryAssetsDir string, kubeConfigPath string, auditLogs bool) (api.ControlPlane, error) {
	vCluster := control.NewControlPlane(binaryAssetsDir, kubeConfigPath, auditLogs)
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
	fs.StringVar(&cfg.kubeConfigPath, "target-kvcl-kubeconfig", defaultKVCLKubeConfigPath, "Path where the kubeconfig file for the virtual cluster is written")
	fs.BoolVar(&cfg.auditLogs, "audit-logs", false, "Enable audit logs for API server")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if err := cfg.resolveBinaryAssetsPath(); err != nil {
		return cfg, err
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
