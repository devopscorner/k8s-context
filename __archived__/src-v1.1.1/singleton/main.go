package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/muesli/termenv"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

const (
	Logo = `
 _    ___                            _            _
| | _( _ ) ___        ___ ___  _ __ | |_ _____  _| |_
| |/ / _ \/ __|_____ / __/ _ \| '_ \| __/ _ \ \/ / __|
|   < (_) \__ \_____| (_| (_) | | | | ||  __/>  <| |_
|_|\_\___/|___/      \___\___/|_| |_|\__\___/_/\_\\__|

`
	AppName = "K8S-CONTEXT"
	VERSION = "v1.1.1"
)

var (
	kubeconfig string
)

type KubeConfig struct {
	Files     []string
	Merged    *api.Config
	Overwrite bool
}

func main() {
	logoStyle := termenv.Style{}.Foreground(termenv.ANSIGreen)
	appNameStyle := termenv.Style{}.Foreground(termenv.ANSIWhite).Bold()

	fmt.Println(logoStyle.Styled(Logo))
	fmt.Println("[[ ", appNameStyle.Styled(AppName), " ]] -", VERSION)
	fmt.Println("=============================")
	GetCommands()
}

// -----------
// menus.go
// -----------
func GetCommands() []*cobra.Command {
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	kc := &KubeConfig{}

	versionCmd := &cobra.Command{
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

			clientset, err := GetClientSet(kubeconfig)
			if err != nil {
				return err
			}

			ctx := context.Background() // add context here
			resource := args[0]

			switch resource {

			case "pods":
			case "po":
				namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				if err != nil {
					return err
				}

				for _, ns := range namespaces.Items {
					fmt.Printf("Namespace: %s\n", ns.Name)
					pods, err := clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
					if err != nil {
						return err
					}

					table := tablewriter.NewWriter(os.Stdout)
					// table.SetHeader([]string{"POD NAME", "READY", "STATUS", "RESTARTS", "AGE", "IMAGE", "NODE", "OWNER KIND", "OWNER NAME", "LABELS"})
					table.SetHeader([]string{
						"POD NAME",
						"READY",
						"STATUS",
						"RESTARTS",
						"AGE",
						"IMAGE",
					})

					table.SetAutoFormatHeaders(false)
					table.SetAutoWrapText(false)

					for _, pod := range pods.Items {
						var containerStatuses []string
						for _, cs := range pod.Status.ContainerStatuses {
							containerStatuses = append(containerStatuses, fmt.Sprintf("%s:%s", cs.Name, strconv.FormatBool(cs.Ready)))
						}
						ready, total := CalculateReadiness(&pod)
						age := HumanReadableDuration(time.Since(pod.ObjectMeta.CreationTimestamp.Time))
						image := strings.Join(GetContainerImages(&pod), ", ")
						// node := pod.Spec.NodeName
						// ownerKind, ownerName := GetOwnerKindAndName(&pod)
						// labels := strings.Join(GetLabels(&pod), ", ")

						table.Append([]string{
							pod.Name,
							fmt.Sprintf("%d/%d", ready, total),
							string(pod.Status.Phase),
							strconv.Itoa(int(pod.Status.ContainerStatuses[0].RestartCount)),
							age,
							image,
							// node,
							// ownerKind,
							// ownerName,
							// labels,
						})
					}
					table.Render()
				}

			case "namespaces":
			case "ns":
				namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				if err != nil {
					return err
				}

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{
					// "NAMESPACE",
					"NAME",
					"STATUS",
					"AGE",
				})
				table.SetAutoFormatHeaders(false)
				table.SetAutoWrapText(false)

				for _, ns := range namespaces.Items {
					// namespace := ns.ObjectMeta.Namespace
					name := ns.ObjectMeta.Name
					status := ns.Status.Phase
					age := HumanReadableDuration(time.Since(ns.ObjectMeta.CreationTimestamp.Time))

					table.Append([]string{
						// namespace,
						name,
						string(status),
						age,
					})
				}
				table.Render()

			case "services":
			case "svc":
				namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				if err != nil {
					return err
				}

				for _, ns := range namespaces.Items {
					fmt.Printf("Namespace: %s\n", ns.Name)
					services, err := clientset.CoreV1().Services(ns.Name).List(ctx, metav1.ListOptions{})
					if err != nil {
						return err
					}

					table := tablewriter.NewWriter(os.Stdout)
					table.SetHeader([]string{"NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)", "AGE"})

					table.SetAutoFormatHeaders(false)
					table.SetAutoWrapText(false)

					for _, service := range services.Items {
						var externalIPs string
						if len(service.Spec.ExternalIPs) > 0 {
							externalIPs = strings.Join(service.Spec.ExternalIPs, ", ")
						} else {
							externalIPs = "<none>"
						}
						age := HumanReadableDuration(time.Since(service.ObjectMeta.CreationTimestamp.Time))
						ports := make([]string, len(service.Spec.Ports))
						for i, port := range service.Spec.Ports {
							ports[i] = fmt.Sprintf("%d/%s", port.Port, string(port.Protocol))
						}

						table.Append([]string{
							service.Name,
							string(service.Spec.Type),
							service.Spec.ClusterIP,
							externalIPs,
							strings.Join(ports, ", "),
							age,
						})
					}
					table.Render()
				}

			default:
				return fmt.Errorf("unknown resource type: %s", resource)
			}

			return nil
		},
	}

	listContextsCmd := &cobra.Command{
		Use:   "list",
		Short: "List the available contexts in the kubeconfig file",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Loading kubeconfig...")
			if err := kc.Load(); err != nil {
				return err
			}

			fmt.Printf("Loaded kubeconfig with %d contexts\n", len(kc.Merged.Contexts))

			var contextNames []string
			for contextName := range kc.Merged.Contexts {
				contextNames = append(contextNames, contextName)
			}

			fmt.Printf("Available contexts: %v\n", contextNames)

			return nil
		},
	}

	selectContextCmd := &cobra.Command{
		Use:   "select",
		Short: "Select a context from the kubeconfig file",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			fmt.Printf("Selected context: %s\n", selectedContext)

			config := kc.Merged
			context, ok := config.Contexts[selectedContext]
			if !ok {
				return fmt.Errorf("context not found: %s", selectedContext)
			}

			cluster, ok := config.Clusters[context.Cluster]
			if !ok {
				return fmt.Errorf("cluster not found: %s", context.Cluster)
			}

			auth, ok := config.AuthInfos[context.AuthInfo]
			if !ok {
				return fmt.Errorf("auth info not found: %s", context.AuthInfo)
			}

			fmt.Printf("Cluster server: %s\n", cluster.Server)
			fmt.Printf("Cluster certificate authority: %s\n", cluster.CertificateAuthority)
			fmt.Printf("User name: %s\n", auth.Username)

			return nil
		},
	}

	switchContextCmd := &cobra.Command{
		Use:   "switch",
		Short: "Switch to a different context",
		RunE: func(cmd *cobra.Command, args []string) error {
			return SwitchContext(kc)
		},
	}

	showContextCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the current context",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ShowContext(kc)
		},
	}

	rootCmd := &cobra.Command{Use: "k8s-context"}
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", kubeconfig, "Path to kubeconfig file")

	rootCmd.AddCommand(versionCmd, loadCmd, mergeCmd, getCmd, listContextsCmd, selectContextCmd, switchContextCmd, showContextCmd)

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("error executing command: %v", err)
	}

	return []*cobra.Command{versionCmd, loadCmd, mergeCmd, getCmd, listContextsCmd, selectContextCmd}
}

// -----------
// context.go
// -----------
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

func GetCurrentContext(config *api.Config) (string, error) {
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

// -----------
// context.go
// -----------
func GetContainerImages(pod *corev1.Pod) []string {
	var images []string
	for _, container := range pod.Spec.Containers {
		images = append(images, container.Image)
	}
	return images
}

func GetOwnerKindAndName(pod *corev1.Pod) (string, string) {
	var ownerKind, ownerName string
	for _, ownerReference := range pod.OwnerReferences {
		ownerKind = string(ownerReference.Kind)
		ownerName = ownerReference.Name
		break
	}
	return ownerKind, ownerName
}

func GetLabels(pod *corev1.Pod) []string {
	var labels []string
	for k, v := range pod.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(labels)
	return labels
}

func HumanReadableDuration(duration time.Duration) string {
	if duration.Seconds() < 60 {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration.Minutes() < 60 {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration.Hours() < 24 {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	return fmt.Sprintf("%dd", int(duration.Hours()/24))
}

func CalculateReadiness(pod *corev1.Pod) (int, int) {
	var ready, total int
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
		total++
	}
	return ready, total
}
