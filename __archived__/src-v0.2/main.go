package main

import (
	"github.com/devopscorner/k8s-context/src/cmd"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "k8s-context",
		Short: "A tool for managing Kubernetes contexts and namespaces",
	}

	for _, cmd := range cmd.GetCommands() {
		rootCmd.AddCommand(cmd)
	}

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
