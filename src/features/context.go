package features

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/AlecAivazis/survey/v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type KubeConfig struct {
	Files     []string
	Merged    *clientcmdapi.Config
	Overwrite bool
}

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

func SwitchContext(kc *KubeConfig) error {
	if err := kc.Load(); err != nil {
		return err
	}

	var contextNames []string
	for contextName := range kc.Merged.Contexts {
		contextNames = append(contextNames, contextName)
	}

	var selectedContext string
	prompt := &survey.Select{
		Message: "Select a context",
		Options: contextNames,
	}

	if err := survey.AskOne(prompt, &selectedContext, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	config := kc.Merged
	_, ok := config.Contexts[selectedContext]
	if !ok {
		return fmt.Errorf("context not found: %s", selectedContext)
	}

	config.CurrentContext = selectedContext
	if err := kc.SaveToFile(kubeconfig); err != nil {
		return err
	}

	fmt.Printf("Switched to context: %s\n", selectedContext)

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

func ChangeKubeconfigContext(kubeconfigPath string, contextName string) error {
	// Load the Kubernetes configuration file.
	kubeconfigBytes, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		return err
	}

	// Parse the configuration file into an API object.
	kubeconfig, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		return err
	}

	// Check if the specified context exists.
	if _, ok := kubeconfig.Contexts[contextName]; !ok {
		return errors.New("context does not exist in the Kubernetes configuration file")
	}

	// Change the current context to the new context.
	kubeconfig.CurrentContext = contextName

	// Write the modified configuration back to the file.
	err = clientcmd.ModifyConfig(clientcmd.NewDefaultPathOptions(), *kubeconfig, true)
	if err != nil {
		return err
	}

	fmt.Printf("\n> Changed context to: %s\n", kubeconfig.CurrentContext)
	return nil
}
