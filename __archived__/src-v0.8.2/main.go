package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

type KubeConfig struct {
	Files     []string
	Merged    *api.Config
	Overwrite bool
}

func (kc *KubeConfig) Load() error {
	var configs []*api.Config
	for _, file := range kc.Files {
		loaded, err := clientcmd.LoadFromFile(file)
		if err != nil {
			return err
		}
		configs = append(configs, loaded)
	}
	merged, err := mergeConfigs(configs)
	if err != nil {
		return err
	}
	kc.Merged = merged
	return nil
}

func (kc *KubeConfig) SaveToFile(file string) error {
	return clientcmd.WriteToFile(*kc.Merged, file)
}

func mergeConfigs(configs []*api.Config) (*api.Config, error) {
	newConfig := &api.Config{
		Kind:       "Config",
		APIVersion: "v1",
		Clusters:   make(map[string]*api.Cluster),
		AuthInfos:  make(map[string]*api.AuthInfo),
		Contexts:   make(map[string]*api.Context),
	}

	for _, config := range configs {
		for k, v := range config.AuthInfos {
			newConfig.AuthInfos[k] = v
		}
		for k, v := range config.Clusters {
			newConfig.Clusters[k] = v
		}
		for k, v := range config.Contexts {
			newConfig.Contexts[k] = v
		}
	}
	for _, config := range configs {
		if config.CurrentContext != "" {
			newConfig.CurrentContext = config.CurrentContext
			break
		}
	}

	return newConfig, nil
}

func getClientset(kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

const (
	VERSION = "v0.5"
)

func main() {
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	kc := &KubeConfig{}

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of k8s-context",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("k8s-context " + VERSION)
		},
	}

	loadCmd := &cobra.Command{
		Use:   "load",
		Short: "Load a kubeconfig file",
		RunE: func(cmd *cobra.Command, args []string) error {
			kc.Files = args
			if err := kc.Load(); err != nil {
				return err
			}
			fmt.Printf("Loaded kubeconfig file(s):\n%s\n", kc.Files)
			return nil
		},
	}

	mergeCmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge multiple kubeconfig files",
		RunE: func(cmd *cobra.Command, args []string) error {
			kc.Files = args
			if err := kc.Load(); err != nil {
				return err
			}
			mergedFile := "merged-config"
			if len(args) > 0 {
				mergedFile = args[0]
			}
			if err := kc.SaveToFile(mergedFile); err != nil {
				return err
			}
			fmt.Printf("Merged kubeconfig files:\n%s\n", kc.Files)
			fmt.Printf("Saved merged kubeconfig to file: %s\n", mergedFile)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get kubernetes resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("resource type not specified")
			}

			clientset, err := getClientset(kubeconfig)
			if err != nil {
				return err
			}

			ctx := context.Background() // add context here
			resource := args[0]
			switch resource {
			case "pods":
				pods, err := clientset.CoreV1().Pods("").List(ctx, v1.ListOptions{})
				if err != nil {
					return err
				}

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Name", "Namespace", "Status"})
				for _, pod := range pods.Items {
					table.Append([]string{pod.Name, pod.Namespace, string(pod.Status.Phase)})
				}
				table.Render()
			default:
				return fmt.Errorf("unknown resource type: %s", resource)
			}

			return nil
		},
	}

	rootCmd := &cobra.Command{Use: "k8s-context"}
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", kubeconfig, "Path to kubeconfig file")

	rootCmd.AddCommand(versionCmd, loadCmd, mergeCmd, getCmd)

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("error executing command: %v", err)
	}
}
