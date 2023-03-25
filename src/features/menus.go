package features

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
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
