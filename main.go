package main

import (
	"flag"
	"os"
	"rancher-kubeconfig-updater/internal/kubeconfig"
	"rancher-kubeconfig-updater/internal/rancher"

	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	var err error

	// Parse command line flags
	autoCreate := flag.Bool("auto-create", false, "Automatically create kubeconfig entries for clusters not found in the config")
	flag.BoolVar(autoCreate, "a", false, "Automatically create kubeconfig entries for clusters not found in the config (shorthand)")
	flag.Parse()

	// Initialize logger with custom config
	logConfig := zap.NewProductionConfig()
	logConfig.Encoding = "console"
	logConfig.DisableCaller = true
	logConfig.DisableStacktrace = true
	logConfig.EncoderConfig.TimeKey = "time"
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	logConfig.EncoderConfig.ConsoleSeparator = " | "
	logger, _ := logConfig.Build()
	defer logger.Sync()

	// Get environment variables
	rancherURL := os.Getenv("RANCHER_URL")
	rancherUsername := os.Getenv("RANCHER_USERNAME")
	rancherPassword := os.Getenv("RANCHER_PASSWORD")

	kubeconfigPath := "~/.kube/config"
	config, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		logger.Error("Failed to load kubeconfig file", zap.Error(err))
		return
	}

	// Check if this is a new config (no users means it's newly created)
	if len(config.Users) == 0 && len(config.Clusters) == 0 && len(config.Contexts) == 0 {
		logger.Info("Creating new kubeconfig file at ~/.kube/config")
	}

	client, err := rancher.NewClient(rancherURL, rancherUsername, rancherPassword, logger)
	if err != nil {
		logger.Error("Failed to authenticate with Rancher", zap.Error(err))
		return
	}

	clusters, err := client.ListClusters()
	if err != nil {
		logger.Error("Failed to retrieve cluster list from Rancher", zap.Error(err))
		return
	}

	for _, v := range clusters {
		clusterToken := client.GetClusterToken(v.ID)
		err = config.UpdateTokenByName(v.ID, v.Name, clusterToken, rancherURL, *autoCreate, logger)
		if err != nil {
			// Error is already logged in UpdateTokenByName
			continue
		}
		logger.Info("Successfully updated kubeconfig token for cluster: " + v.Name)
	}

	err = config.SaveKubeconfig(kubeconfigPath)
	if err != nil {
		logger.Error("Failed to save kubeconfig file", zap.Error(err))
		return
	}

	logger.Info("All cluster tokens have been updated successfully")
}
