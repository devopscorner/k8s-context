package features

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	VERSION = "v1.1.3"
)

var (
	kubeconfig string
)

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

	getCmd.Flags().StringSlice("namespace", []string{}, "Namespaces to filter resources by (comma-separated)")

	rootCmd := &cobra.Command{Use: "k8s-context"}
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", kubeconfig, "Path to kubeconfig file")

	rootCmd.AddCommand(versionCmd, loadCmd, mergeCmd, getCmd, listContextsCmd, selectContextCmd, switchContextCmd, showContextCmd)

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("error executing command: %v", err)
	}

	return []*cobra.Command{versionCmd, loadCmd, mergeCmd, getCmd, listContextsCmd, selectContextCmd}
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
