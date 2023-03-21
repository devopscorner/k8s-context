package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

const (
	VERSION = "v0.5"
)

func GetCommands() []*cobra.Command {
	var kubeconfigPath string

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of k8s-tool",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("k8s-tool " + VERSION)
		},
	}

	var listContextsCmd = &cobra.Command{
		Use:   "list-contexts",
		Short: "List all available contexts",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := loadKubeconfig(kubeconfigPath)
			if err != nil {
				fmt.Printf("Error loading kubeconfig: %v\n", err)
				os.Exit(1)
			}

			for contextName := range config.Contexts {
				fmt.Println(contextName)
			}
		},
	}

	var loadFromPathCmd = &cobra.Command{
		Use:   "load-from-path",
		Short: "Load kubeconfig from a specific path",
		Run: func(cmd *cobra.Command, args []string) {
			config, err := loadKubeconfig(kubeconfigPath)
			if err != nil {
				fmt.Printf("Error loading kubeconfig: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Loaded kubeconfig from %s\n", kubeconfigPath)
			for contextName := range config.Contexts {
				fmt.Println(contextName)
			}
		},
	}

	var mergeKubeconfigCmd = &cobra.Command{
		Use:   "merge-kubeconfig",
		Short: "Merge multiple kubeconfig files",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			configPaths := args

			configs := make([]*api.Config, len(configPaths))
			for i, path := range configPaths {
				config, err := loadKubeconfig(path)
				if err != nil {
					fmt.Printf("Error loading kubeconfig from %s: %v\n", path, err)
					os.Exit(1)
				}
				configs[i] = config
			}
			mergedConfig := mergeKubeconfigs(configs)

			for contextName := range mergedConfig.Contexts {
				fmt.Println(contextName)
			}
		},
	}

	var saveMergedKubeconfigCmd = &cobra.Command{
		Use:   "save-merged-kubeconfig",
		Short: "Save the merged kubeconfig to a file",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			configPaths := args

			configs := make([]*api.Config, len(configPaths))
			for i, path := range configPaths {
				config, err := loadKubeconfig(path)
				if err != nil {
					fmt.Printf("Error loading kubeconfig from %s: %v\n", path, err)
					os.Exit(1)
				}
				configs[i] = config
			}
			mergedConfig := mergeKubeconfigs(configs)

			err := clientcmd.WriteToFile(*mergedConfig, kubeconfigPath)
			if err != nil {
				fmt.Printf("Error saving merged kubeconfig: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Merged kubeconfig saved to %s\n", kubeconfigPath)
		},
	}

	var selectContextCmd = &cobra.Command{
		Use:   "select-context",
		Short: "Interactively select a context from all kubeconfig files in the ~/.kube/ directory",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			kubeconfigFiles, err := getKubeconfigFiles()
			if err != nil {
				fmt.Printf("Error finding kubeconfig files: %v\n", err)
				os.Exit(1)
			}

			configs := make([]*api.Config, len(kubeconfigFiles))
			for i, path := range kubeconfigFiles {
				config, err := loadKubeconfig(path)
				if err != nil {
					fmt.Printf("Error loading kubeconfig from %s: %v\n", path, err)
					os.Exit(1)
				}
				configs[i] = config
			}
			mergedConfig := mergeKubeconfigs(configs)

			contextNames := make([]string, 0, len(mergedConfig.Contexts))
			for name := range mergedConfig.Contexts {
				contextNames = append(contextNames, name)
			}

			var selectedContext string
			prompt := &survey.Select{
				Message: "Choose a context:",
				Options: contextNames,
			}
			err = survey.AskOne(prompt, &selectedContext)
			if err != nil {
				fmt.Printf("Error selecting context: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Selected context: %s\n", selectedContext)
		},
	}

	return []*cobra.Command{versionCmd, listContextsCmd, loadFromPathCmd, mergeKubeconfigCmd, saveMergedKubeconfigCmd, selectContextCmd}
}

func loadKubeconfig(path string) (*api.Config, error) {
	return clientcmd.LoadFromFile(path)
}

func mergeKubeconfigs(configs []*api.Config) *api.Config {
	mergedConfig := api.NewConfig()
	for _, config := range configs {
		for contextName, context := range config.Contexts {
			mergedConfig.Contexts[contextName] = context
		}
		for clusterName, cluster := range config.Clusters {
			mergedConfig.Clusters[clusterName] = cluster
		}
		for userName, user := range config.AuthInfos {
			mergedConfig.AuthInfos[userName] = user
		}
	}
	return mergedConfig
}

func getKubeconfigFiles() ([]string, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	kubeDir := filepath.Join(usr.HomeDir, ".kube")
	fileList, err := os.ReadDir(kubeDir)
	if err != nil {
		return nil, err
	}

	kubeconfigFiles := []string{}
	for _, file := range fileList {
		if !file.IsDir() && strings.HasSuffix(file.Name(), "config") {
			kubeconfigFiles = append(kubeconfigFiles, filepath.Join(kubeDir, file.Name()))
		}
	}

	return kubeconfigFiles, nil
}
