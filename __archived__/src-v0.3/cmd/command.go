package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	kubeconfig     string
	kubeconfigList []string
	mergeConfigs   string
	savePath       string
)

const (
	VERSION = "v0.5"
)

func GetCommands() []*cobra.Command {
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of k8s-context",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("k8s-context " + VERSION)
		},
	}

	return []*cobra.Command{versionCmd}
}

func switchContext() {
	configPath := filepath.Join(homedir.HomeDir(), ".kube", "config")

	if kubeconfig != "" {
		configPath = kubeconfig
	}

	if mergeConfigs != "" {
		configPath = mergeKubeconfigs(strings.Split(mergeConfigs, ","))
	}

	if len(kubeconfigList) > 0 {
		configPath = mergeKubeconfigs(kubeconfigList)
	}

	if savePath != "" {
		err := copyFile(configPath, savePath)
		if err != nil {
			fmt.Printf("Error saving merged kubeconfig: %v\n", err)
			os.Exit(1)
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		fmt.Printf("Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating kubernetes client: %v\n", err)
		os.Exit(1)
	}

	displayContexts(clientset, config)
}

func mergeKubeconfigs(paths []string) string {
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

func displayContexts(clientset *kubernetes.Clientset, config *rest.Config) {
	contexts := config.Contexts

	for contextName, context := range contexts {
		fmt.Printf("Context: %s\n", contextName)
		fmt.Printf("Cluster: %s\n", context.Cluster)

		podList, err := clientset.CoreV1().Pods("").List(v1.ListOptions{})
		if err != nil {
			fmt.Printf("Error listing pods: %v\n", err)
			continue
		}

		fmt.Printf("POD NAME\tREADY\tSTATUS\tRESTARTS\tAGE\tIMAGE\tNODE\tOWNER KIND\tOWNER NAME\tLABELS\n")
		for _, pod := range podList.Items {
			ready := fmt.Sprintf("%d/%d", pod.ReplicationControllerStatus.ReadyReplicas, pod.ReplicationControllerStatus.AvailableReplicas)
			age := time.Since(pod.GetCreationTimestamp().Time)
			ownerKind, ownerName := getPodOwner(pod)
			labels := labels.Set(pod.GetLabels())

			fmt.Printf("%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
				pod.Name,
				ready,
				pod.Status.Phase,
				pod.ContainerStatus.RestartCount,
				age,
				pod.Spec.Containers[0].Image,
				pod.Spec.NodeName,
				ownerKind,
				ownerName,
				labels,
			)
		}
		fmt.Println()
	}
}

func getPodOwner(pod v1.Pod) (string, string) {
	for _, ownerRef := range pod.GetOwnerReferences() {
		if ownerRef.Controller != nil && *ownerRef.Controller {
			return ownerRef.Kind, ownerRef.Name
		}
	}

	return "", ""
}

func copyFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer output.Close()

	_, err = io.Copy(output, input)
	if err != nil {
		return err
	}

	return nil
}
