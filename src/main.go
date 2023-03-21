package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

var (
	kubeconfigList []string
	mergeConfigs   string
	savePath       string
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
	restarts  int32
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
		// Use default kubeconfig file if no files are specified
		kubeconfigFiles = []string{filepath.Join(homedir.HomeDir(), ".kube", "config")}
	}

	configs := make([]*api.Config, len(kubeconfigFiles))

	for i, kubeconfigFile := range kubeconfigFiles {
		config, err := clientcmd.LoadFromFile(kubeconfigFile)
		if err != nil {
			return nil, err
		}
		configs[i] = config
	}

	config, err := mergeKubeconfigs(configs)
	if err != nil {
		return nil, err
	}

	// Set the current context to the first context if it's not already set
	if config.CurrentContext == "" && len(config.Contexts) > 0 {
		for contextName := range config.Contexts {
			config.CurrentContext = contextName
			break
		}
	}

	return config, nil
}

func mergeKubeconfigs(paths []string) (*api.Config, error) {
	loadingRules := clientcmd.ClientConfigLoadingRules{}

	mergedConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&loadingRules,
		&clientcmd.ConfigOverrides{},
	)

	for _, path := range paths {
		loadingRules.Precedence = append(loadingRules.Precedence, path)
	}

	config, err := mergedConfig.RawConfig()
	if err != nil {
		fmt.Printf("Error merging kubeconfig files: %v\n", err)
		os.Exit(1)
	}

	mergedPath := filepath.Join(homedir.HomeDir(), ".kube", "merged_config")
	err = clientcmd.WriteToFile(config, mergedPath)
	if err != nil {
		fmt.Printf("Error writing merged kubeconfig: %v\n", err)
		os.Exit(1)
	}

	return mergedPath
}

func writeKubeconfigToFile(config *api.Config, outputPath string) error {
	kubeconfigBytes, err := clientcmd.Write(*config)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(outputPath, kubeconfigBytes, 0644)
}

func getContextsFromConfig(config *api.Config) []string {
	contexts := make([]string, 0, len(config.Contexts))
	for contextName := range config.Contexts {
		contexts = append(contexts, contextName)
	}
	return contexts
}

func selectContextInteractively(contexts []string) (string, error) {
	fmt.Println("Available contexts:")
	for i, contextName := range contexts {
		fmt.Printf("[%d] %s\n", i+1, contextName)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Select context number: ")
		text, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		text = strings.TrimSpace(text)
		index, err := strconv.Atoi(text)
		if err != nil || index < 1 || index > len(contexts) {
			fmt.Printf("Invalid selection: %s\n", text)
			continue
		}

		return contexts[index-1], nil
	}
}

func getContextInfo(config *api.Config, contextName string) (*contextInfo, error) {
	context := config.Contexts[contextName]
	cluster := config.Clusters[context.Cluster]
	authInfo := config.AuthInfos[context.AuthInfo]

	clientConfig := &rest.Config{
		Host: cluster.Server,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure:   cluster.InsecureSkipTLSVerify,
			ServerName: cluster.Server,
			CertData:   authInfo.ClientCertificateData,
			KeyData:    authInfo.ClientKeyData,
			CAData:     cluster.CertificateAuthorityData,
		},
		Username: authInfo.Username,
		Password: authInfo.Password,
		BearerToken: func() string {
			if authInfo.Token != "" {
				return authInfo.Token
			}
			if authInfo.TokenFile != "" {
				tokenBytes, err := ioutil.ReadFile(authInfo.TokenFile)
				if err != nil {
					return ""
				}
				return string(tokenBytes)
			}
			return ""
		}(),
		QPS:   1000.0,
		Burst: 2000,
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	namespace := context.Namespace
	if namespace == "" {
		namespace = "default"
	}

	clientConfigRaw, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw config for context %s: %v", contextName, err)
	}

	clientset, err := kubernetes.NewForConfig(clientConfigRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client for context %s: %v", contextName, err)
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(clientcmd.ContextWithNoCache(clientConfigRaw), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pods := make([]podInfo, len(podList.Items))
	for i, pod := range podList.Items {
		containerStatus := pod.Status.ContainerStatuses[0]
		podAge := time.Since(pod.ObjectMeta.CreationTimestamp.Time).Truncate(time.Second).String()

		pods[i] = podInfo{
			name:      pod.ObjectMeta.Name,
			ready:     strconv.FormatBool(containerStatus.Ready),
			status:    string(pod.Status.Phase),
			restarts:  containerStatus.RestartCount,
			age:       podAge,
			image:     containerStatus.Image,
			node:      pod.Spec.NodeName,
			ownerKind: "",
			ownerName: "",
			labels:    labelsToString(pod.ObjectMeta.Labels),
		}

		ownerReferences := pod.ObjectMeta.OwnerReferences
		if len(ownerReferences) > 0 {
			ownerRef := ownerReferences[0]
			pods[i].ownerKind = ownerRef.Kind
			pods[i].ownerName = ownerRef.Name
		}
	}

	return &contextInfo{
		name:      contextName,
		cluster:   cluster.Server,
		apiServer: cluster.Server,
		username:  authInfo.Username,
		namespace: namespace,
		pods:      pods,
	}, nil
}

func printPodsTable(pods []podInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"POD NAME", "READY", "STATUS", "RESTARTS", "AGE", "IMAGE", "NODE", "OWNER KIND", "OWNER NAME", "LABELS"})
	for _, pod := range pods {
		table.Append([]string{
			pod.name,
			pod.ready,
			pod.status,
			strconv.FormatInt(int64(pod.restarts), 10),
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

func labelsToString(labels map[string]string) string {
	var sb strings.Builder
	for key, value := range labels {
		sb.WriteString(fmt.Sprintf("%s=%s,", key, value))
	}
	return strings.TrimRight(sb.String(), ",")
}
