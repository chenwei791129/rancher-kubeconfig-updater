package main

import (
	"os"
	"rancher-kubeconfig-updater/cmd"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	rootCmd := cmd.NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
