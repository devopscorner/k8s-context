package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/muesli/termenv"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
	VERSION = "v1.1.6"
)

var (
	kubeconfig     string
	loadFile       string
	selectedConfig string
	configBytes    []byte
	err            error
)

type KubeConfig struct {
	Files     []string
	Merged    *clientcmdapi.Config
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

// -------------------------------------------------------------------
// pods.go
// -------------------------------------------------------------------
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

func ShowPodsByFilter(pods *corev1.PodList) {
	table := tablewriter.NewWriter(os.Stdout)
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

		table.Append([]string{
			pod.Name,
			fmt.Sprintf("%d/%d", ready, total),
			string(pod.Status.Phase),
			strconv.Itoa(int(pod.Status.ContainerStatuses[0].RestartCount)),
			age,
			image,
		})
	}
	table.Render()
}

func ShowNamespaceByFilter(namespaces *corev1.NamespaceList) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"NAME",
		"STATUS",
		"AGE",
	})
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)

	for _, ns := range namespaces.Items {
		name := ns.ObjectMeta.Name
		status := ns.Status.Phase
		age := HumanReadableDuration(time.Since(ns.ObjectMeta.CreationTimestamp.Time))

		table.Append([]string{
			name,
			string(status),
			age,
		})
	}
	table.Render()
}

func ShowServiceByFilter(services *corev1.ServiceList) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"NAME",
		"TYPE",
		"CLUSTER-IP",
		"EXTERNAL-IP",
		"PORT(S)",
		"AGE",
	})

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

func ShowDeploymentByFilter(deployments *v1.DeploymentList) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"NAME",
		"READY",
		"UP-TO-DATE",
		"AVAILABLE",
		"AGE",
	})

	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)

	for _, deploy := range deployments.Items {
		name := deploy.Name
		age := HumanReadableDuration(time.Since(deploy.ObjectMeta.CreationTimestamp.Time))

		table.Append([]string{
			name,
			fmt.Sprintf("%d/%d", deploy.Status.ReadyReplicas, deploy.Status.Replicas),
			fmt.Sprintf("%d", deploy.Status.UpdatedReplicas),
			fmt.Sprintf("%d", deploy.Status.AvailableReplicas),
			age,
		})
	}
	table.Render()
}

// -------------------------------------------------------------------
// menus.go
// -------------------------------------------------------------------
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

	listContextsCmd := &cobra.Command{
		Use:   "list",
		Short: "List all available Kubernetes contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			InitConfig()

			if configBytes == nil {
				// Print the list of context names
				fmt.Println("No available contexts!")
			} else {
				// Get the map of context name to context config
				config, err := clientcmd.Load(configBytes)
				if err != nil {
					return err
				}
				contextsMap := config.Contexts

				// Print the list of context names
				fmt.Println("Available Kubernetes contexts:")
				for contextName := range contextsMap {
					fmt.Println(contextName)
				}
			}
			return nil
		},
	}

	loadCmd := &cobra.Command{
		Use:   "load [file...]",
		Short: "Load a kubeconfig file",
		Long:  "Load one or more kubeconfig files into k8s-context",
		RunE: func(cmd *cobra.Command, args []string) error {
			var files []string
			if len(args) == 0 {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				kubeDir := filepath.Join(home, ".kube")
				err = filepath.Walk(kubeDir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() && strings.HasPrefix(info.Name(), "config") {
						files = append(files, path)
					}
					return nil
				})
				if err != nil {
					return err
				}
			} else {
				files = args
			}

			// Prompt the user to select a kubeconfig file if more than one is found
			if len(files) > 1 {
				var selected string
				prompt := &survey.Select{
					Message: "Select a kubeconfig file:",
					Options: files,
				}
				if err := survey.AskOne(prompt, &selected); err != nil {
					return err
				}
				files = []string{selected}
			}

			kc.Files = files
			if err := kc.Load(); err != nil {
				return err
			}
			selectedConfig = strings.Join(kc.Files, "\n")
			fmt.Printf("Loaded kubeconfig file(s):\n%s\n", selectedConfig)

			// Get the map of context name to context config
			configBytes, err = os.ReadFile(selectedConfig)
			config, err := clientcmd.Load(configBytes)
			if err != nil {
				return err
			}
			var contextNames []string
			for contextName := range config.Contexts {
				contextNames = append(contextNames, contextName)
			}

			SelectedConfig(contextNames, config)
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
		Short: "Get Kubernetes resources (ns, svc, deploy, po)",
		Long:  "Get Kubernetes resources: namespace (ns), services (svc), deployments (deploy), pods (po)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("resource type not specified")
			}

			clientset, err := GetClientSet(kubeconfig)
			if err != nil {
				return err
			}

			ctx := context.Background()
			resource := args[0]

			namespaces, err := cmd.Flags().GetStringSlice("namespace")
			if err != nil {
				return err
			}

			if len(namespaces) == 0 {
				// If namespace is not specified, get all namespaces
				nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				if err != nil {
					return err
				}
				for _, ns := range nsList.Items {
					namespace := ns.Name
					fmt.Printf("Namespace: %s\n", namespace)
					switch resource {

					case "pods", "po":
						pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
						if err != nil {
							return err
						}
						ShowPodsByFilter(pods)

					case "namespaces", "ns":
						var namespaces *corev1.NamespaceList
						if namespace != "" {
							ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
							if err != nil {
								return err
							}
							namespaces = &corev1.NamespaceList{Items: []corev1.Namespace{*ns}}
						} else {
							ns, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
							if err != nil {
								return err
							}
							namespaces = ns
						}
						ShowNamespaceByFilter(namespaces)

					case "services", "svc":
						services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
						if err != nil {
							return err
						}
						ShowServiceByFilter(services)

					case "deployment", "deploy":
						deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
						if err != nil {
							return err
						}
						ShowDeploymentByFilter(deployments)

					default:
						return fmt.Errorf("unknown resource type: %s", resource)
					}
				}
			} else {
				// If namespace is specified, get resources only in those namespaces
				for _, namespace := range namespaces {
					fmt.Printf("Namespace: %s\n", namespace)
					switch resource {
					case "pods", "po":
						pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
						if err != nil {
							return err
						}
						ShowPodsByFilter(pods)

					case "namespaces", "ns":
						var namespaces *corev1.NamespaceList
						if namespace != "" {
							ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
							if err != nil {
								return err
							}
							namespaces = &corev1.NamespaceList{Items: []corev1.Namespace{*ns}}
						} else {
							ns, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
							if err != nil {
								return err
							}
							namespaces = ns
						}
						ShowNamespaceByFilter(namespaces)

					case "services", "svc":
						services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
						if err != nil {
							return err
						}
						ShowServiceByFilter(services)

					case "deployment", "deploy":
						deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
						if err != nil {
							return err
						}
						ShowDeploymentByFilter(deployments)

					default:
						return fmt.Errorf("unknown resource type: %s", resource)
					}
				}
			}

			return nil
		},
	}

	switchContextCmd := &cobra.Command{
		Use:   "switch",
		Short: "Switch to different context",
		RunE: func(cmd *cobra.Command, args []string) error {
			InitConfig()

			// Get the map of context name to context config
			config, err := clientcmd.Load(configBytes)
			if err != nil {
				return err
			}
			var contextNames []string
			for contextName := range config.Contexts {
				contextNames = append(contextNames, contextName)
			}

			SelectedConfig(contextNames, config)
			return nil
		},
	}

	getCmd.Flags().StringSlice("namespace", []string{}, "Namespaces to filter resources by (comma-separated)")
	listContextsCmd.Flags().StringVarP(&loadFile, "file", "f", "", "Using spesific kubeconfig file")
	switchContextCmd.Flags().StringVarP(&loadFile, "file", "f", "", "Using spesific kubeconfig file")

	rootCmd := &cobra.Command{Use: "k8s-context"}
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", kubeconfig, "Path to kubeconfig file")

	rootCmd.AddCommand(versionCmd, getCmd, listContextsCmd, loadCmd, mergeCmd, switchContextCmd)

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("error executing command: %v", err)
	}

	return []*cobra.Command{versionCmd, getCmd, listContextsCmd, loadCmd, mergeCmd, switchContextCmd}
}

// -------------------------------------------------------------------
// context.go
// -------------------------------------------------------------------
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
