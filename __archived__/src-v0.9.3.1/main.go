package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/devopscorner/k8s-context/src/features"
	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/homedir"
)

const (
	VERSION = "v0.5"
)

var (
	kubeconfig string
	err        error
)

func main() {
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	kc := &features.KubeConfig{}

	var versionCmd = &cobra.Command{
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

			clientset, err := features.GetClientSet(kubeconfig)
			if err != nil {
				return err
			}

			ctx := context.Background() // add context here
			resource := args[0]

			switch resource {

			case "pods":
				namespaces, err := clientset.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
				if err != nil {
					return err
				}

				for _, ns := range namespaces.Items {
					fmt.Printf("Namespace: %s\n", ns.Name)
					pods, err := clientset.CoreV1().Pods(ns.Name).List(ctx, v1.ListOptions{})
					if err != nil {
						return err
					}

					table := tablewriter.NewWriter(os.Stdout)
					// table.SetHeader([]string{"POD NAME", "READY", "STATUS", "RESTARTS", "AGE", "IMAGE", "NODE", "OWNER KIND", "OWNER NAME", "LABELS"})
					table.SetHeader([]string{"POD NAME", "READY", "STATUS", "RESTARTS", "AGE", "IMAGE"})

					table.SetAutoFormatHeaders(false)
					table.SetAutoWrapText(false)

					for _, pod := range pods.Items {
						var containerStatuses []string
						for _, cs := range pod.Status.ContainerStatuses {
							containerStatuses = append(containerStatuses, fmt.Sprintf("%s:%s", cs.Name, strconv.FormatBool(cs.Ready)))
						}
						ready, total := features.CalculateReadiness(&pod)
						age := features.HumanReadableDuration(time.Since(pod.ObjectMeta.CreationTimestamp.Time))
						image := strings.Join(features.GetContainerImages(&pod), ", ")
						// node := pod.Spec.NodeName
						// ownerKind, ownerName := features.GetOwnerKindAndName(&pod)
						// labels := strings.Join(features.GetLabels(&pod), ", ")

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
				namespaces, err := clientset.CoreV1().Namespaces().List(ctx, v1.ListOptions{})
				if err != nil {
					return err
				}

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"NAMESPACE"})
				table.SetAutoFormatHeaders(false)
				table.SetAutoWrapText(false)

				for _, ns := range namespaces.Items {
					table.Append([]string{ns.Name})
				}
				table.Render()

			default:
				return fmt.Errorf("unknown resource type: %s", resource)
			}

			return nil
		},
	}

	listContextsCmd := &cobra.Command{
		Use:   "list-contexts",
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
		Use:   "select-context",
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

	rootCmd := &cobra.Command{Use: "k8s-context"}
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", kubeconfig, "Path to kubeconfig file")

	rootCmd.AddCommand(versionCmd, loadCmd, mergeCmd, getCmd, listContextsCmd, selectContextCmd)

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("error executing command: %v", err)
	}
}
