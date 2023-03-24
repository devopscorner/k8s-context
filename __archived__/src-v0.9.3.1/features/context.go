package features

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
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
	merged, err := MergeConfigs(configs)
	if err != nil {
		return err
	}
	kc.Merged = merged
	return nil
}

// func (kc *KubeConfig) Load() error {
// 	fmt.Printf("Loading kubeconfig from files: %v\n", kc.Files)
// 	var configs []*api.Config
// 	for _, file := range kc.Files {
// 		loaded, err := clientcmd.LoadFromFile(file)
// 		if err != nil {
// 			return err
// 		}
// 		configs = append(configs, loaded)
// 	}
// 	merged, err := MergeConfigs(configs)
// 	if err != nil {
// 		return err
// 	}
// 	kc.Merged = merged

// 	fmt.Printf("Loaded kubeconfig with %d contexts\n", len(kc.Merged.Contexts))
// 	for contextName := range kc.Merged.Contexts {
// 		fmt.Printf("Context: %s\n", contextName)
// 	}

// 	return nil
// }

func (kc *KubeConfig) SaveToFile(file string) error {
	return clientcmd.WriteToFile(*kc.Merged, file)
}

func MergeConfigs(configs []*api.Config) (*api.Config, error) {
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

func GetCurrentContext() (string, error) {
	var kubeconfig string

	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return "", err
	}
	if config.CurrentContext == "" {
		return "", fmt.Errorf("no current context set in kubeconfig file")
	}
	return config.CurrentContext, nil
}
