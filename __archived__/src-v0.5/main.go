package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

type kubeconfig struct {
	name string
	path string
}

type contextInfo struct {
	name      string
	cluster   string
	apiServer string
	username  string
	namespace string
	pods      []podInfo
}

type podInfo struct {
	name      string
	ready     string
	status    string
	restarts  string
	age       string
	image     string
	node      string
	ownerKind string
	ownerName string
	labels    string
}

func main() {
	var kubeconfigFiles []string
	var outputKubeconfigFile string

	rootCmd := &cobra.Command{
		Use:   "k8s-context",
		Short: "Kubernetes (k8s) change context tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := mergeKubeconfigFiles(kubeconfigFiles)
			if err != nil {
				return err
			}

			if outputKubeconfigFile != "" {
				if err := writeKubeconfigToFile(config, outputKubeconfigFile); err != nil {
					return err
				}
				fmt.Printf("Merged kubeconfig saved to file %s\n", outputKubeconfigFile)
			}

			contexts := getContextsFromConfig(config)
			selectedContext, err := selectContextInteractively(contexts)
			if err != nil {
				return err
			}

			contextInfo, err := getContextInfo(config, selectedContext)
			if err != nil {
				return err
			}

			fmt.Printf("Selected context: %s\n", selectedContext)
			fmt.Printf("Cluster: %s\n", contextInfo.cluster)
			fmt.Printf("API Server: %s\n", contextInfo.apiServer)
			fmt.Printf("Username: %s\n", contextInfo.username)
			fmt.Printf("Namespace: %s\n", contextInfo.namespace)

			if len(contextInfo.pods) > 0 {
				fmt.Printf("Pods in namespace %s:\n", contextInfo.namespace)
				printPodsTable(contextInfo.pods)
			}

			return nil
		},
	}

	rootCmd.Flags().StringSliceVarP(&kubeconfigFiles, "kubeconfig", "k", []string{}, "Paths to kubeconfig files to merge (can be specified multiple times)")
	rootCmd.Flags().StringVarP(&outputKubeconfigFile, "output", "o", "", "Output path for merged kubeconfig file")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func mergeKubeconfigFiles(kubeconfigFiles []string) (*api.Config, error) {
	if len(kubeconfigFiles) == 0 {
		// Use default kubeconfig file if none is specified
		kubeconfigFiles = []string{filepath.Join(homedir.HomeDir(), ".kube", "config")}
	}

	var mergedConfig *api.Config

	for _, kubeconfigFile := range kubeconfigFiles {
		data, err := ioutil.ReadFile(kubeconfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read kubeconfig file %s: %v", kubeconfigFile, err)
		}

		config, err := clientcmd.Load(data)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig file %s: %v", kubeconfigFile, err)
		}

		if mergedConfig == nil {
			// Use first kubeconfig file as base for merged config
			mergedConfig = config
		} else {
			// Merge current config into merged config
			mergedConfig, err = mergeKubeconfigs(mergedConfig, config)
			if err != nil {
				return nil, err
			}
		}
	}

	return mergedConfig, nil
}

func mergeKubeconfigs(config1 *api.Config, config2 *api.Config) (*api.Config, error) {
	data1, err := clientcmd.Write(*config1)
	if err != nil {
		return nil, fmt.Errorf("failed to write kubeconfig 1: %v", err)
	}

	data2, err := clientcmd.Write(*config2)
	if err != nil {
		return nil, fmt.Errorf("failed to write kubeconfig 2: %v", err)
	}

	mergedData, err := strategicpatch.StrategicMergePatch(data1, data2, api.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to merge kubeconfigs: %v", err)
	}

	mergedConfig, err := clientcmd.Load(mergedData)
	if err != nil {
		return nil, fmt.Errorf("failed to load merged kubeconfig: %v", err)
	}

	return mergedConfig, nil
}

func writeKubeconfigToFile(config *api.Config, outputKubeconfigFile string) error {
	data, err := clientcmd.Write(*config)
	if err != nil {
		return fmt.Errorf("failed to write merged kubeconfig: %v", err)
	}
	if err := ioutil.WriteFile(outputKubeconfigFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write merged kubeconfig to file %s: %v", outputKubeconfigFile, err)
	}

	return nil
}

func getContextsFromConfig(config *api.Config) []string {
	contexts := []string{}

	for contextName := range config.Contexts {
		contexts = append(contexts, contextName)
	}

	return contexts
}

func selectContextInteractively(contexts []string) (string, error) {
	if len(contexts) == 0 {
		return "", fmt.Errorf("no contexts found in kubeconfig file(s)")
	}

	if len(contexts) == 1 {
		return contexts[0], nil
	}

	fmt.Println("Select a context:")

	for i, context := range contexts {
		fmt.Printf("%d. %s\n", i+1, context)
	}

	var selection int

	if _, err := fmt.Scanln(&selection); err != nil {
		return "", fmt.Errorf("failed to read selection: %v", err)
	}

	if selection < 1 || selection > len(contexts) {
		return "", fmt.Errorf("invalid selection")
	}

	return contexts[selection-1], nil
}

func getContextInfo(config *api.Config, contextName string) (*contextInfo, error) {
	context := config.Contexts[contextName]
	clusterName := context.Cluster
	if clusterName == "" {
		return nil, fmt.Errorf("no cluster found for context %s", contextName)
	}

	cluster, found := config.Clusters[clusterName]
	if !found {
		return nil, fmt.Errorf("cluster %s not found for context %s", clusterName, contextName)
	}
	apiServer := cluster.Server

	var username string
	if context.AuthInfo != "" {
		authInfo, found := config.AuthInfos[context.AuthInfo]
		if !found {
			return nil, fmt.Errorf("auth info %s not found for context %s", context.AuthInfo, contextName)
		}

		username = authInfo.Username
	}

	var namespace string
	if context.Namespace != "" {
		namespace = context.Namespace
	}

	clientConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{
		CurrentContext: contextName,
	})

	clientConfigRaw, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw config for context %s: %v", contextName, err)
	}

	clientset, err := kubernetes.NewForConfig(clientConfigRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client for context %s: %v", contextName, err)
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(clientcmd.(clientConfigRaw), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %v", namespace, err)
	}

	podInfos := []podInfo{}
	for _, pod := range pods.Items {
		podInfo := podInfo{
			name:      pod.ObjectMeta.Name,
			ready:     pod.Status.ContainerStatuses[0].Ready,
			status:    pod.Status.Phase,
			restarts:  fmt.Sprintf("%d", pod.ContainerStatuses[0].RestartCount),
			age:       getAge(pod.CreationTimestamp.Time),
			image:     pod.Spec.Containers[0].Image,
			node:      pod.Spec.NodeName,
			ownerKind: getOwnerKind(pod.OwnerReferences),
			ownerName: getOwnerName(pod.OwnerReferences),
			labels:    getLabelsString(pod.ObjectMeta.Labels),
		}

		podInfos = append(podInfos, podInfo)
	}

	contextInfo := &contextInfo{
		name:      contextName,
		cluster:   clusterName,
		apiServer: apiServer,
		username:  username,
		namespace: namespace,
		pods:      podInfos,
	}

	return contextInfo, nil
}

func getAge(creationTime time.Time) string {
	now := time.Now()
	duration := now.Sub(creationTime)

	var age string

	if duration.Hours() > 24 {
		days := int(duration.Hours() / 24)
		age = fmt.Sprintf("%d days", days)
	} else {
		age = fmt.Sprintf("%.2f hours", duration.Hours())
	}

	return age
}

func getOwnerKind(ownerReferences []metav1.OwnerReference) string {
	if len(ownerReferences) == 0 {
		return ""
	}

	return ownerReferences[0].Kind
}

func getOwnerName(ownerReferences []metav1.OwnerReference) string {
	if len(ownerReferences) == 0 {
		return ""
	}
	return ownerReferences[0].Name
}

func getLabelsString(labels map[string]string) string {
	var labelsString string
	for key, value := range labels {
		labelsString += fmt.Sprintf("%s=%s,", key, value)
	}

	if len(labelsString) > 0 {
		labelsString = labelsString[:len(labelsString)-1]
	}

	return labelsString
}

func printPodsTable(pods []podInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"POD NAME", "READY", "STATUS", "RESTARTS", "AGE", "IMAGE", "NODE", "OWNER KIND", "OWNER NAME", "LABELS"})

	for _, pod := range pods {
		table.Append([]string{
			pod.name,
			pod.ready,
			pod.status,
			pod.restarts,
			pod.age,
			pod.image,
			pod.node,
			pod.ownerKind,
			pod.ownerName,
			pod.labels,
		})
	}

	table.Render()
}
