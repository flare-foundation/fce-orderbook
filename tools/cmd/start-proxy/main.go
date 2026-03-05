// TEMPORARY: This command starts the extension proxy as a Go process.
// It will be replaced by a Docker container once the Dockerfile is implemented.
// See EXTENSION-TEMPLATE-SPEC.md §5 for the Docker approach.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"extension-e2e/pkg/utils"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	proxyConfig "github.com/flare-foundation/tee-proxy/pkg/config"
	initProxy "github.com/flare-foundation/tee-proxy/pkg/init"
	"github.com/joho/godotenv"
)

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	ctx := context.TODO()

	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: Error loading .env file: %v\n", err)
	}

	proxyConfigFile := findProxyConfig()

	initProxy.Init(ctx, proxyConfigFile)
	logger.Infof("Started extension proxy")

	err := logProxyAndTeeIds(proxyConfigFile)
	if err != nil {
		logger.Warnf("Failed to log proxy and tee IDs: %v", err)
	}

	sig := <-signalChan
	logger.Infof("Received %v signal, shutting down", sig)
}

func findProxyConfig() string {
	// Try project-root relative path first.
	_, thisFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	candidate := filepath.Join(projectRoot, "config", "proxy", "extension_proxy.toml")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	// Fallback to working directory relative.
	return "./config/proxy/extension_proxy.toml"
}

func logProxyAndTeeIds(configFile string) error {
	config, err := proxyConfig.Read(configFile)
	if err != nil {
		return fmt.Errorf("failed to read proxy config: %w", err)
	}

	proxyURL := fmt.Sprintf("http://localhost:%s", config.Ports.External)

	teeID, proxyID, err := utils.GetTeeProxyID(proxyURL)
	if err != nil {
		return fmt.Errorf("failed to extract teeID and proxyID: %w", err)
	}

	logger.Infof("Proxy started - TeeID: %s, ProxyID: %s", teeID.Hex(), proxyID.Hex())
	return nil
}
