package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"unmarshall/kvcl/api"
	"unmarshall/kvcl/pkg/control"
	"unmarshall/kvcl/pkg/util"
)

func main() {
	defer util.OnExit()
	var (
		vCluster api.ControlPlane
		err      error
	)
	ctx := setupSignalHandler()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	binaryAssetsPath, err := parseCmdArgs()
	if err != nil {
		util.ExitAppWithError(1, fmt.Errorf("failed to get binary assets path: %w", err))
	}
	logger.Info("starting virtual cluster", "binaryAssetsPath", binaryAssetsPath)
	defer func() {
		logger.Info("shutting down virtual cluster...")
		if err = vCluster.Stop(); err != nil {
			logger.Error("failed to stop virtual cluster", "error", err)
		}
	}()
	vCluster, err = startVirtualCluster(ctx, binaryAssetsPath)
	if err != nil {
		util.ExitAppWithError(1, fmt.Errorf("failed to start virtual cluster: %w", err))
	}
	<-ctx.Done()
}

func startVirtualCluster(ctx context.Context, binaryAssetsPath string) (api.ControlPlane, error) {
	vCluster := control.NewControlPlane(binaryAssetsPath)
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

func parseCmdArgs() (string, error) {
	var binaryAssetsPath string
	args := os.Args[1:]
	fs := flag.CommandLine
	fs.StringVar(&binaryAssetsPath, "binary-assets-path", "", "Path to the binary assets")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if binaryAssetsPath == "" {
		binaryAssetsPath = getBinaryAssetsPathFromEnv()
	}
	if binaryAssetsPath == "" {
		return "", fmt.Errorf("cannot find binary-assets-path")
	}
	return binaryAssetsPath, nil
}

func getBinaryAssetsPathFromEnv() string {
	return os.Getenv("BINARY_ASSETS_DIR")
}
