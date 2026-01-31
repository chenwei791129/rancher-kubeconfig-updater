// Package cmd implements the command-line interface for the rancher-kubeconfig-updater application.
package cmd

import (
	"os"
	"rancher-kubeconfig-updater/internal/config"
	"rancher-kubeconfig-updater/internal/kubeconfig"
	"rancher-kubeconfig-updater/internal/logger"
	"rancher-kubeconfig-updater/internal/rancher"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd/api"
)

var (
	autoCreate            bool
	authTypeFlag          string
	userFlag              string
	passwordFlag          string
	clusterFlag           string
	insecureSkipTLSVerify bool
	configPath            string
	thresholdDays         int
	forceRefresh          bool
	dryRun                bool
	withDirectly          bool
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
	rootCmd.Flags().BoolVar(&insecureSkipTLSVerify, "insecure-skip-tls-verify", false, "Skip TLS certificate verification (insecure, use only for development/testing)")
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to kubeconfig file (default: ~/.kube/config)")
	rootCmd.Flags().IntVar(&thresholdDays, "threshold-days", 30, "Expiration threshold in days")
	rootCmd.Flags().BoolVar(&forceRefresh, "force-refresh", false, "Bypass expiration checks and force regeneration")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without modifying kubeconfig")
	rootCmd.Flags().BoolVar(&withDirectly, "with-directly", false, "Include Downstream Directly contexts for direct cluster access")

	return rootCmd
}

func run(cmd *cobra.Command, args []string) {
	var err error

	// Initialize logger with pipe-delimited format
	zapLogger := logger.NewLogger()
	defer func() {
		_ = zapLogger.Sync()
	}()

	// Get configuration with priority: Flag > Env > Default
	rancherURL := os.Getenv("RANCHER_URL")
	rancherUsername := config.GetConfig(cmd, "user", "RANCHER_USERNAME")
	rancherAuthType := config.GetConfig(cmd, "auth-type", "RANCHER_AUTH_TYPE")
	insecureSkipTLSVerify := config.GetBool(cmd, "insecure-skip-tls-verify", "RANCHER_INSECURE_SKIP_TLS_VERIFY")
	thresholdDays := config.GetInt(cmd, "threshold-days", "TOKEN_THRESHOLD_DAYS")
	forceRefresh := config.GetBool(cmd, "force-refresh", "FORCE_REFRESH")
	dryRun := config.GetBool(cmd, "dry-run", "DRY_RUN")
	withDirectly := config.GetBool(cmd, "with-directly", "WITH_DIRECTLY")

	// Log dry-run mode if enabled
	if dryRun {
		zapLogger.Info("[DRY-RUN] Mode enabled - no changes will be made to kubeconfig")
	}

	// Log with-directly mode if enabled
	if withDirectly {
		zapLogger.Info("Downstream Directly mode enabled - will include direct cluster contexts")
	}

	rancherPassword, err := config.GetPassword(cmd, "password", "RANCHER_PASSWORD")
	if err != nil {
		zapLogger.Error("Failed to read password", zap.Error(err))
		return
	}

	// Use the configPath from the flag if provided, otherwise use empty string for default
	// Empty string will automatically resolve to ~/.kube/config on Unix/macOS and %USERPROFILE%\.kube\config on Windows
	kubecfg, err := kubeconfig.LoadKubeconfig(configPath)
	if err != nil {
		zapLogger.Error("Failed to load kubeconfig file", zap.Error(err))
		return
	}

	// Check if this is a new config (no users means it's newly created)
	if len(kubecfg.AuthInfos) == 0 && len(kubecfg.Clusters) == 0 && len(kubecfg.Contexts) == 0 {
		zapLogger.Info("Creating new kubeconfig file at default location")
	}

	// Determine auth type
	authType := rancher.AuthTypeLocal
	if rancherAuthType == "ldap" {
		authType = rancher.AuthTypeLDAP
	} else if rancherAuthType == "local" {
		authType = rancher.AuthTypeLocal
	} else if rancherAuthType != "" {
		zapLogger.Error("Invalid auth-type value. Must be 'local' or 'ldap'")
		return
	}

	client, err := rancher.NewClient(rancherURL, rancherUsername, rancherPassword, authType, zapLogger, insecureSkipTLSVerify)
	if err != nil {
		zapLogger.Error("Failed to authenticate with Rancher", zap.Error(err))
		return
	}

	clusters, err := client.ListClusters()
	if err != nil {
		zapLogger.Error("Failed to retrieve cluster list from Rancher", zap.Error(err))
		return
	}

	// Filter clusters if --cluster flag is specified
	if clusterFlag != "" {
		clusters = filterClusters(clusters, clusterFlag, zapLogger)
	}

	// Track dry-run statistics
	var clustersToUpdate, clustersToSkip int

	for _, v := range clusters {
		// Get current token from kubeconfig if it exists
		var currentToken string
		if authInfo, exists := kubecfg.AuthInfos[v.Name]; exists {
			currentToken = authInfo.Token
		}

		// Determine if token regeneration is needed
		decision := client.DetermineTokenRegeneration(currentToken, forceRefresh, thresholdDays, v.Name)

		// Log decision and skip if regeneration not needed
		logTokenDecision(zapLogger, decision, v.Name, dryRun)

		if !decision.ShouldRegenerate {
			clustersToSkip++
			continue
		}

		clustersToUpdate++

		// Skip actual token regeneration and kubeconfig update in dry-run mode
		if dryRun {
			continue
		}

		// Get full kubeconfig from Rancher (includes Downstream Directly contexts if available)
		clusterKubeconfig, err := client.GetClusterKubeconfig(v.ID)
		if err != nil {
			zapLogger.Error("Failed to get kubeconfig for cluster",
				zap.String("cluster", v.Name),
				zap.Error(err))
			continue
		}

		// Check if we should use the new merge approach or legacy approach
		if withDirectly || autoCreate {
			// Use MergeKubeconfig for new approach (supports Downstream Directly)
			kubeconfig.MergeKubeconfig(kubecfg, clusterKubeconfig, v.Name, withDirectly)
			if withDirectly {
				// Count direct contexts for logging
				directCount := countDirectContexts(clusterKubeconfig, v.Name)
				if directCount > 0 {
					zapLogger.Info("Successfully updated kubeconfig with direct contexts",
						zap.String("cluster", v.Name),
						zap.Int("directContexts", directCount))
				} else {
					zapLogger.Info("Successfully updated kubeconfig token for cluster: " + v.Name)
				}
			} else {
				zapLogger.Info("Successfully updated kubeconfig token for cluster: " + v.Name)
			}
		} else {
			// Legacy approach: extract token and update only if user exists
			var token string
			for _, authInfo := range clusterKubeconfig.AuthInfos {
				if authInfo.Token != "" {
					token = authInfo.Token
					break
				}
			}
			err = kubeconfig.UpdateTokenByName(kubecfg, v.ID, v.Name, token, rancherURL, autoCreate, zapLogger)
			if err != nil {
				// Error is already logged in UpdateTokenByName
				continue
			}
			zapLogger.Info("Successfully updated kubeconfig token for cluster: " + v.Name)
		}
	}

	// Skip saving in dry-run mode and show summary
	if dryRun {
		zapLogger.Info("[DRY-RUN] Summary",
			zap.Int("clustersToUpdate", clustersToUpdate),
			zap.Int("clustersToSkip", clustersToSkip))
		zapLogger.Info("[DRY-RUN] No changes were made to kubeconfig")
		return
	}

	err = kubeconfig.SaveKubeconfig(kubecfg, configPath, zapLogger)
	if err != nil {
		zapLogger.Error("Failed to save kubeconfig file", zap.Error(err))
		return
	}

	zapLogger.Info("All cluster tokens have been updated successfully")
}

// logTokenDecision logs the token regeneration decision with consistent formatting
func logTokenDecision(logger *zap.Logger, decision rancher.TokenRegenerationDecision, clusterName string, dryRun bool) {
	if !decision.ShouldRegenerate {
		// Log skip decisions
		if dryRun {
			logger.Info("[DRY-RUN] Would skip token regeneration",
				zap.String("cluster", clusterName),
				zap.String("reason", string(decision.Reason)),
				zap.Float64("daysUntilExpiration", decision.DaysUntilExpiry))
		} else {
			switch decision.Reason {
			case rancher.ReasonNeverExpires:
				logger.Info("Token never expires, skipping regeneration",
					zap.String("cluster", clusterName))
			case rancher.ReasonStillValid:
				logger.Info("Token is still valid, skipping regeneration",
					zap.String("cluster", clusterName),
					zap.String("expiresAt", decision.ExpiresAt.Format("2006-01-02 15:04:05")),
					zap.Int("daysUntilExpiration", int(decision.DaysUntilExpiry)))
			}
		}
		return
	}

	// Log regeneration decisions
	if dryRun {
		logger.Info("[DRY-RUN] Would regenerate token",
			zap.String("cluster", clusterName),
			zap.String("reason", string(decision.Reason)),
			zap.Float64("daysUntilExpiration", decision.DaysUntilExpiry))
	} else {
		switch decision.Reason {
		case rancher.ReasonForceRefreshEnabled:
			logger.Info("Force refresh enabled, regenerating token",
				zap.String("cluster", clusterName))
		case rancher.ReasonNoExistingToken:
			logger.Info("No existing token, generating new token",
				zap.String("cluster", clusterName))
		case rancher.ReasonExpiresSoon:
			logger.Info("Token expires soon, regenerating",
				zap.String("cluster", clusterName),
				zap.String("expiresAt", decision.ExpiresAt.Format("2006-01-02 15:04:05")),
				zap.Int("daysUntilExpiration", int(decision.DaysUntilExpiry)))
		case rancher.ReasonNeverExpiresButRefreshRequired:
			logger.Info("Regenerating token (never expires but refresh required)",
				zap.String("cluster", clusterName))
		case rancher.ReasonExpirationCheckFailed:
			logger.Info("Regenerating token due to expiration check failure",
				zap.String("cluster", clusterName))
		}
	}
}

// filterClusters filters clusters based on comma-separated cluster names or IDs
func filterClusters(clusters rancher.Clusters, clusterFilter string, logger *zap.Logger) rancher.Clusters {
	// Parse comma-separated cluster names/IDs and create a set for fast lookup
	// Overall complexity: O(n) where n is the number of clusters
	allowedClustersRaw := strings.Split(clusterFilter, ",")
	allowedClustersSet := make(map[string]struct{})

	// Trim whitespace and convert to lowercase for case-insensitive matching
	for _, c := range allowedClustersRaw {
		trimmed := strings.TrimSpace(c)
		if trimmed != "" {
			allowedClustersSet[strings.ToLower(trimmed)] = struct{}{}
		}
	}

	if len(allowedClustersSet) == 0 {
		logger.Warn("--cluster flag specified but no valid cluster names provided, processing all clusters")
		return clusters
	}

	// Filter clusters
	filteredClusters := make(rancher.Clusters, 0)
	addedClusterIDs := make(map[string]struct{})
	matchedFilters := make(map[string]struct{})

	for _, cluster := range clusters {
		// Skip if this cluster was already added
		if _, added := addedClusterIDs[cluster.ID]; added {
			continue
		}

		// Check if cluster name or ID matches any of the allowed clusters (case-insensitive)
		clusterNameLower := strings.ToLower(cluster.Name)
		clusterIDLower := strings.ToLower(cluster.ID)

		nameMatches := false
		idMatches := false

		if _, exists := allowedClustersSet[clusterNameLower]; exists {
			nameMatches = true
		}
		if _, exists := allowedClustersSet[clusterIDLower]; exists {
			idMatches = true
		}

		if nameMatches || idMatches {
			filteredClusters = append(filteredClusters, cluster)
			// Record all matched filters (both name and ID if they both match)
			// to prevent false "not found" warnings
			if nameMatches {
				matchedFilters[clusterNameLower] = struct{}{}
			}
			if idMatches {
				matchedFilters[clusterIDLower] = struct{}{}
			}
			addedClusterIDs[cluster.ID] = struct{}{}
		}
	}

	// Log warnings for clusters not found
	for allowed := range allowedClustersSet {
		if _, matched := matchedFilters[allowed]; !matched {
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

// countDirectContexts counts the number of Downstream Directly contexts in a kubeconfig
// Direct contexts are identified by having a name that starts with "{clusterName}-"
func countDirectContexts(cfg *api.Config, clusterName string) int {
	count := 0
	prefix := clusterName + "-"
	for ctxName := range cfg.Contexts {
		if strings.HasPrefix(ctxName, prefix) {
			count++
		}
	}
	return count
}
