package main

import (
	"fmt"
	"os"

	"github.com/devopscorner/k8s-context/src/cmd"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "k8s-context",
		Short: "Switch Kubernetes context",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.switchContext()
		},
	}

	rootCmd.Flags().StringVarP(&cmd.kubeconfig, "kubeconfig", "k", "", "Path to the kubeconfig file")
	rootCmd.Flags().StringSliceVarP(&cmd.kubeconfigList, "kubeconfig-list", "l", []string{}, "List of kubeconfig files to merge")
	rootCmd.Flags().StringVarP(&cmd.mergeConfigs, "merge-configs", "m", "", "Merge multiple kubeconfig files (comma-separated list)")
	rootCmd.Flags().StringVarP(&cmd.savePath, "save-path", "s", "", "Save the merged kubeconfig to a file")

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
