package cmd

import (
	"os"
	"rancher-kubeconfig-updater/internal/config"
	"rancher-kubeconfig-updater/internal/kubeconfig"
	"rancher-kubeconfig-updater/internal/rancher"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	autoCreate   bool
	authTypeFlag string
	userFlag     string
	passwordFlag string
	clusterFlag  string
)

func NewRootCmd() *cobra.Command {
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
	rootCmd.Flags().StringVar(&clusterFlag, "cluster", "", "Comma-separated list of cluster names or IDs to update")

	return rootCmd
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
	defer func() {
		_ = logger.Sync()
	}()

	// Get configuration with priority: Flag > Env > Default
	rancherURL := os.Getenv("RANCHER_URL")
	rancherUsername := config.GetConfig(cmd, "user", "RANCHER_USERNAME")
	rancherAuthType := config.GetConfig(cmd, "auth-type", "RANCHER_AUTH_TYPE")

	rancherPassword, err := config.GetPassword(cmd, "password", "RANCHER_PASSWORD")
	if err != nil {
		logger.Error("Failed to read password", zap.Error(err))
		return
	}

	// Use empty string to let expandPath use the default platform-specific path
	// This will automatically resolve to ~/.kube/config on Unix/macOS and %USERPROFILE%\.kube\config on Windows
	kubeconfigPath := ""
	kubecfg, err := kubeconfig.LoadKubeconfig(kubeconfigPath)
	if err != nil {
		logger.Error("Failed to load kubeconfig file", zap.Error(err))
		return
	}

	// Check if this is a new config (no users means it's newly created)
	if len(kubecfg.AuthInfos) == 0 && len(kubecfg.Clusters) == 0 && len(kubecfg.Contexts) == 0 {
		logger.Info("Creating new kubeconfig file at default location")
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

	// Filter clusters if --cluster flag is specified
	if clusterFlag != "" {
		clusters = filterClusters(clusters, clusterFlag, logger)
	}

	for _, v := range clusters {
		clusterToken := client.GetClusterToken(v.ID)
		err = kubeconfig.UpdateTokenByName(kubecfg, v.ID, v.Name, clusterToken, rancherURL, autoCreate, logger)
		if err != nil {
			// Error is already logged in UpdateTokenByName
			continue
		}
		logger.Info("Successfully updated kubeconfig token for cluster: " + v.Name)
	}

	err = kubeconfig.SaveKubeconfig(kubecfg, kubeconfigPath, logger)
	if err != nil {
		logger.Error("Failed to save kubeconfig file", zap.Error(err))
		return
	}

	logger.Info("All cluster tokens have been updated successfully")
}

// filterClusters filters clusters based on comma-separated cluster names or IDs
func filterClusters(clusters rancher.Clusters, clusterFilter string, logger *zap.Logger) rancher.Clusters {
	// Parse comma-separated cluster names/IDs
	allowedClustersRaw := strings.Split(clusterFilter, ",")
	allowedClusters := make([]string, 0, len(allowedClustersRaw))
	
	// Trim whitespace and convert to lowercase for case-insensitive matching
	for _, c := range allowedClustersRaw {
		trimmed := strings.TrimSpace(c)
		if trimmed != "" {
			allowedClusters = append(allowedClusters, strings.ToLower(trimmed))
		}
	}
	
	if len(allowedClusters) == 0 {
		logger.Warn("--cluster flag specified but no valid cluster names provided, processing all clusters")
		return clusters
	}
	
	// Filter clusters
	filteredClusters := make(rancher.Clusters, 0)
	matchedClusters := make(map[string]bool)
	
	for _, cluster := range clusters {
		// Check if cluster name or ID matches any of the allowed clusters (case-insensitive)
		clusterNameLower := strings.ToLower(cluster.Name)
		clusterIDLower := strings.ToLower(cluster.ID)
		
		for _, allowed := range allowedClusters {
			if clusterNameLower == allowed || clusterIDLower == allowed {
				filteredClusters = append(filteredClusters, cluster)
				matchedClusters[allowed] = true
				break
			}
		}
	}
	
	// Log warnings for clusters not found
	for _, allowed := range allowedClusters {
		if !matchedClusters[allowed] {
			logger.Warn("Specified cluster not found in Rancher", zap.String("cluster", allowed))
		}
	}
	
	if len(filteredClusters) == 0 {
		logger.Warn("No clusters matched the specified filter, no clusters will be updated")
	} else {
		logger.Info("Filtering clusters based on --cluster flag", 
			zap.Int("matched", len(filteredClusters)), 
			zap.Int("total", len(clusters)))
	}
	
	return filteredClusters
}
