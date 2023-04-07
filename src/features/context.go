package features

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/olekukonko/tablewriter"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

type KubeConfig struct {
	Files     []string
	Merged    *clientcmdapi.Config
	Overwrite bool
}

var (
	kubeconfig     string
	loadFile       string
	selectedConfig string
	configBytes    []byte
	err            error
)

func (kc *KubeConfig) Load() error {
	var configs []*clientcmdapi.Config
	for _, file := range kc.Files {
		loaded, err := clientcmd.LoadFromFile(file)
		if err != nil {
			return err
		}
		configs = append(configs, loaded)
	}
	merged, err := MergeConfigs(configs)
	if err != nil {
		return err
	}
	kc.Merged = merged
	return nil
}

func (kc *KubeConfig) SaveToFile(file string) error {
	return clientcmd.WriteToFile(*kc.Merged, file)
}

func MergeConfigs(configs []*clientcmdapi.Config) (*clientcmdapi.Config, error) {
	newConfig := &clientcmdapi.Config{
		Kind:       "Config",
		APIVersion: "v1",
		Clusters:   make(map[string]*clientcmdapi.Cluster),
		AuthInfos:  make(map[string]*clientcmdapi.AuthInfo),
		Contexts:   make(map[string]*clientcmdapi.Context),
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

func GetClientSet(kubeconfig string) (*kubernetes.Clientset, error) {
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

func GetCurrentContext(config *clientcmdapi.Config) (string, error) {
	if config == nil {
		return "", fmt.Errorf("kubeconfig is nil")
	}
	if config.CurrentContext == "" {
		return "", fmt.Errorf("no current context set in kubeconfig")
	}
	context, ok := config.Contexts[config.CurrentContext]
	if !ok {
		return "", fmt.Errorf("current context not found in kubeconfig: %s", config.CurrentContext)
	}
	return context.Cluster, nil
}

func ListContexts(kc *KubeConfig) error {
	fmt.Println("Available contexts:")
	currentContext, err := GetCurrentContext(kc.Merged)
	if err != nil {
		return err
	}
	for contextName := range kc.Merged.Contexts {
		prefix := " "
		if contextName == currentContext {
			prefix = "*"
		}
		fmt.Printf("%s %s\n", prefix, contextName)
	}
	return nil
}

func ShowDetailList(config *clientcmdapi.Config) error {
	contextsMap := config.Contexts

	// Create a slice of context information
	var contextInfo []struct {
		ContextName string
		ClusterName string
	}

	// Iterate through each context and extract the cluster and user information
	for contextName, contextConfig := range contextsMap {
		clusterName := contextConfig.Cluster
		clusterConfig, found := config.Clusters[clusterName]
		if !found {
			return fmt.Errorf("cluster %s not found in config", clusterName)
		}
		contextInfo = append(contextInfo, struct {
			ContextName string
			ClusterName string
		}{
			ContextName: contextName,
			ClusterName: clusterConfig.Server,
		})
	}

	// Print the list of context names
	fmt.Println("Available Kubernetes contexts:")

	// Print the table of context information
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Context Name", "Cluster Name"})
	for _, info := range contextInfo {
		table.Append([]string{info.ContextName, info.ClusterName})
	}
	table.Render()

	return nil
}

func ShowContext(kc *KubeConfig) error {
	if err := kc.Load(); err != nil {
		return err
	}

	currentContext, err := GetCurrentContext(kc.Merged)
	if err != nil {
		return err
	}

	fmt.Printf("Current context: %s\n", currentContext)

	return nil
}

func InitConfig() error {
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")

	// Use default kubeconfig file if load flag is not provided
	if loadFile == "" {
		if selectedConfig != "" {
			configBytes, err = os.ReadFile(selectedConfig)
		} else {
			configBytes, err = os.ReadFile(kubeconfig)
		}
		if err != nil {
			return err
		}
	} else {
		// Load kubeconfig file from flag
		configBytes, err = os.ReadFile(loadFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func SelectedConfig(contextNames []string, config *clientcmdapi.Config) error {
	var selectedContext string
	prompt := &survey.Select{
		Message: "Select a context",
		Options: contextNames,
	}

	if err := survey.AskOne(prompt, &selectedContext, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	fmt.Printf("Selected context: %s\n", selectedContext)

	context, ok := config.Contexts[selectedContext]
	if !ok {
		return fmt.Errorf("context not found: %s", selectedContext)
	}

	cluster, ok := config.Clusters[context.Cluster]
	if !ok {
		return fmt.Errorf("cluster not found: %s", context.Cluster)
	}

	// auth, ok := config.AuthInfos[context.AuthInfo]
	// if !ok {
	// 	return fmt.Errorf("auth info not found: %s", context.AuthInfo)
	// }

	fmt.Printf("Cluster server: %s\n", cluster.Server)
	// fmt.Printf("Cluster certificate authority: %s\n", cluster.CertificateAuthority)
	// fmt.Printf("User name: %s\n", auth.Username)

	if loadFile == "" {
		ChangeKubeconfigContext(kubeconfig, context.Cluster)
	} else {
		ChangeKubeconfigContext(loadFile, context.Cluster)
	}

	return nil
}

func ChangeKubeconfigContext(kubeconfigPath string, contextName string) error {
	// Load the Kubernetes configuration file.
	kubeconfigBytes, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		return err
	}

	// Parse the configuration file into an API object.
	kubeconfig, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		fmt.Printf("\n> Can't read Kubernetes configuration file")
		return err
	}

	// Check if the specified context exists.
	if _, ok := kubeconfig.Contexts[contextName]; !ok {
		fmt.Printf("\n> Context does not exist in the Kubernetes configuration file ($HOME/.kube/config) \n> Merge into your Kubernetes config file first... ")
		return err
	}

	// Change the current context to the new context.
	kubeconfig.CurrentContext = contextName

	// Write the modified configuration back to the file.
	err = clientcmd.ModifyConfig(clientcmd.NewDefaultPathOptions(), *kubeconfig, true)
	if err != nil {
		fmt.Printf("\n> Failed to change context: %s\n", kubeconfig.CurrentContext)
		return err
	} else {
		fmt.Printf("\n> Successfully change context to: %s\n", kubeconfig.CurrentContext)
		return nil
	}
	return nil
}
