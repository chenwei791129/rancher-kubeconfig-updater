// Package config handles application configuration and environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// GetConfig returns the value of a flag if it was set, otherwise returns the value of the environment variable.
func GetConfig(cmd *cobra.Command, flagName, envKey string) string {
	if cmd.Flags().Changed(flagName) {
		val, _ := cmd.Flags().GetString(flagName)
		return val
	}
	return os.Getenv(envKey)
}

// GetPassword returns the password from the flag or environment variable.
// If the flag is set to "-", it prompts the user for the password securely.
func GetPassword(cmd *cobra.Command, flagName, envKey string) (string, error) {
	if cmd.Flags().Changed(flagName) {
		val, _ := cmd.Flags().GetString(flagName)
		if val == "-" {
			fmt.Print("Enter Rancher Password: ")
			bytePassword, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println() // Newline after input
			if err != nil {
				return "", err
			}
			return string(bytePassword), nil
		}
		return val, nil
	}
	return os.Getenv(envKey), nil
}

// GetBool returns the value of a boolean flag if it was set, otherwise returns the value from the environment variable.
func GetBool(cmd *cobra.Command, flagName, envKey string) bool {
	if cmd.Flags().Changed(flagName) {
		val, _ := cmd.Flags().GetBool(flagName)
		return val
	}
	// Check environment variable (case-insensitive)
	envVal := os.Getenv(envKey)
	if envVal == "" {
		return false
	}
	boolVal, err := strconv.ParseBool(envVal)
	if err != nil {
		return false
	}
	return boolVal
}
