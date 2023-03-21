package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func main() {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "A tool for managing Kubernetes contexts",
	}

	loadCmd := &cobra.Command{
		Use:   "load",
		Short: "Load a specific kubeconfig file",
		RunE:  loadKubeconfig,
	}

	mergeCmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge multiple kubeconfig files",
		RunE:  mergeKubeconfigs,
	}

	saveCmd := &cobra.Command{
		Use:   "save",
		Short: "Save merged kubeconfig to a file",
		RunE:  saveKubeconfig,
	}

	cmd.AddCommand(loadCmd, mergeCmd, saveCmd)

	if err := cmd.Execute(); err != nil {
		log.Errorf("Error executing command: %s", err)
		os.Exit(1)
	}
}

func loadKubeconfig(cmd *cobra.Command, args []string) error {
	log.Info("Starting loadKubeconfig")
	defer log.Info("Ending loadKubeconfig")

	kubeconfigPath, err := survey.AskOne(&survey.Input{
		Message: "Enter the path to the kubeconfig file:", nil})
	if err != nil {
		log.Errorf("Error prompting user for kubeconfig path: %s", err)
		return err
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath.(string))
	if err != nil {
		log.Errorf("Error building config from flags: %s", err)
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("Error creating clientset: %s", err)
		return err
	}

	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("Error getting list of pods: %s", err)
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Namespace", "Status"})
	for _, pod := range pods.Items {
		table.Append([]string{pod.Name, pod.Namespace, string(pod.Status.Phase)})
	}
	table.Render()

	return nil
}

func mergeKubeconfigs(cmd *cobra.Command, args []string) error {
	log.Info("Starting mergeKubeconfigs")
	defer log.Info("Ending mergeKubeconfigs")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Errorf("Error getting home directory: %s", err)
		return err
	}

	defaultKubeconfigPath := filepath.Join(homeDir, ".kube", "config")
	kubeconfigPaths := []string{defaultKubeconfigPath}

	for {
		choice, err := survey.AskOne(&survey.Confirm{
			Message: "Do you want to merge another kubeconfig file?",
			Default: false,
		})
		if err != nil {
			log.Errorf("Error prompting user for merge confirmation: %s", err)
			return err
		}

		if !choice.(bool) {
			break
		}

		kubeconfigPath, err := survey.AskOne(&survey.Input{
			Message: "Enter the path to the kubeconfig file:",
		})
		if err != nil {
			log.Errorf("Error prompting user for kubeconfig path: %s", err)
			return err
		}

		kubeconfigPaths = append(kubeconfigPaths, kubeconfigPath.(string))
	}

	configs := []*api.Config{}

	for _, kubeconfigPath := range kubeconfigPaths {
		config, err := clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			log.Errorf("Error loading config from file: %s", err)
			return err
		}

		configs = append(configs, config)
	}

	mergedConfig, err := api.MergeConfig(configs...)
	if err != nil {
		log.Errorf("Error merging configs: %s", err)
		return err
	}

	log.Info("Merged kubeconfig:")

	data, err := clientcmd.Write(*mergedConfig)
	if err != nil {
		log.Errorf("Error writing merged config: %s", err)
		return err
	}

	fmt.Println(string(data))

	return nil
}

func saveKubeconfig(cmd *cobra.Command, args []string) error {
	log.Info("Starting saveKubeconfig")
	defer log.Info("Ending saveKubeconfig")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Errorf("Error getting home directory: %s", err)
		return err
	}

	defaultKubeconfigPath := filepath.Join(homeDir, ".kube", "config")
	kubeconfigPath, err := survey.AskOne(&survey.Input{
		Message: fmt.Sprintf("Enter the path to save the merged kubeconfig (default: %s):", defaultKubeconfigPath),
		Default: defaultKubeconfigPath,
	}, nil)
	if err != nil {
		log.Errorf("Error prompting user for kubeconfig path: %s", err)
		return err
	}

	configs := []*api.Config{}

	for _, arg := range args {
		config, err := clientcmd.LoadFromFile(arg)
		if err != nil {
			log.Errorf("Error loading config from file: %s", err)
			return err
		}

		configs = append(configs, config)
	}

	mergedConfig, err := api.MergeConfig(configs...)
	if err != nil {
		log.Errorf("Error merging configs: %s", err)
		return err
	}

	err = clientcmd.WriteToFile(*mergedConfig, kubeconfigPath.(string))
	if err != nil {
		log.Errorf("Error writing merged config to file: %s", err)
		return err
	}

	fmt.Printf("Merged kubeconfig saved to %s\n", kubeconfigPath)

	return nil
}
