package main

import (
	"os"
	"rancher-kubeconfig-updater/internal/config"
	"rancher-kubeconfig-updater/internal/kubeconfig"
	"rancher-kubeconfig-updater/internal/rancher"

	_ "github.com/joho/godotenv/autoload"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	autoCreate   bool
	authTypeFlag string
	userFlag     string
	passwordFlag string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "rancher-kubeconfig-updater",
		Short: "Update kubeconfig tokens for Rancher-managed Kubernetes clusters",
		Run:   run,
	}

	rootCmd.Flags().BoolVarP(&autoCreate, "auto-create", "a", false, "Automatically create kubeconfig entries for clusters not found in the config")
	rootCmd.Flags().StringVar(&authTypeFlag, "auth-type", "", "Authentication type: 'local' or 'ldap' (default: from RANCHER_AUTH_TYPE env or 'local')")
	rootCmd.Flags().StringVarP(&userFlag, "user", "u", "", "Rancher Username")
	rootCmd.Flags().StringVarP(&passwordFlag, "password", "p", "", "Rancher Password")
	// Set NoOptDefVal for password to allow interactive prompt when flag is present without value
	rootCmd.Flags().Lookup("password").NoOptDefVal = "-"

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	var err error

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

	// Get configuration with priority: Flag > Env > Default
	rancherURL := os.Getenv("RANCHER_URL")
	rancherUsername := config.GetConfig(cmd, "user", "RANCHER_USERNAME")
	rancherAuthType := config.GetConfig(cmd, "auth-type", "RANCHER_AUTH_TYPE")

	rancherPassword, err := config.GetPassword(cmd, "password", "RANCHER_PASSWORD")
	if err != nil {
		logger.Error("Failed to read password", zap.Error(err))
		return
	}

	kubeconfigPath := "~/.kube/config"
	kubecfg, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		logger.Error("Failed to load kubeconfig file", zap.Error(err))
		return
	}

	// Check if this is a new config (no users means it's newly created)
	if len(kubecfg.Users) == 0 && len(kubecfg.Clusters) == 0 && len(kubecfg.Contexts) == 0 {
		logger.Info("Creating new kubeconfig file at ~/.kube/config")
	}

	// Determine auth type
	authType := rancher.AuthTypeLocal
	if rancherAuthType == "ldap" {
		authType = rancher.AuthTypeLDAP
	} else if rancherAuthType == "local" {
		authType = rancher.AuthTypeLocal
	} else if rancherAuthType != "" {
		logger.Error("Invalid auth-type value. Must be 'local' or 'ldap'")
		return
	}

	client, err := rancher.NewClient(rancherURL, rancherUsername, rancherPassword, authType, logger)
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
		err = kubecfg.UpdateTokenByName(v.ID, v.Name, clusterToken, rancherURL, autoCreate, logger)
		if err != nil {
			// Error is already logged in UpdateTokenByName
			continue
		}
		logger.Info("Successfully updated kubeconfig token for cluster: " + v.Name)
	}

	err = kubecfg.SaveKubeconfig(kubeconfigPath)
	if err != nil {
		logger.Error("Failed to save kubeconfig file", zap.Error(err))
		return
	}

	logger.Info("All cluster tokens have been updated successfully")
}
