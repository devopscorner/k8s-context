package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
				pods, err := clientset.CoreV1().Pods("").List(ctx, v1.ListOptions{})
				if err != nil {
					return err
				}

				table := tablewriter.NewWriter(os.Stdout)
				// table.SetHeader([]string{"POD NAME", "NAMESPACE", "READY", "STATUS", "RESTARTS", "AGE", "IMAGE", "NODE", "OWNER KIND", "OWNER NAME", "LABELS"})
				// table.SetHeader([]string{"POD NAME", "NAMESPACE", "STATUS", "RESTARTS", "AGE", "IMAGE", "NODE", "OWNER KIND", "OWNER NAME", "LABELS"})
				table.SetHeader([]string{"POD NAME", "NAMESPACE", "STATUS", "RESTARTS", "AGE", "IMAGE"})

				table.SetAutoFormatHeaders(false)
				table.SetAutoWrapText(false)

				for _, pod := range pods.Items {
					var containerStatuses []string
					for _, cs := range pod.Status.ContainerStatuses {
						containerStatuses = append(containerStatuses, fmt.Sprintf("%s:%s", cs.Name, strconv.FormatBool(cs.Ready)))
					}
					// ready, total := features.CalculateReadiness(&pod)
					age := features.HumanReadableDuration(time.Since(pod.ObjectMeta.CreationTimestamp.Time))
					image := strings.Join(features.GetContainerImages(&pod), ", ")
					// node := pod.Spec.NodeName
					// ownerKind, ownerName := features.GetOwnerKindAndName(&pod)
					// labels := strings.Join(features.GetLabels(&pod), ", ")

					table.Append([]string{
						pod.Name,
						// fmt.Sprintf("%d/%d", ready, total),
						pod.Namespace,
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

			default:
				return fmt.Errorf("unknown resource type: %s", resource)
			}

			return nil
		},
	}

	rootCmd := &cobra.Command{Use: "k8s-context"}
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", kubeconfig, "Path to kubeconfig file")

	rootCmd.AddCommand(versionCmd, loadCmd, mergeCmd, getCmd)

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("error executing command: %v", err)
	}
}
