package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/devopscorner/k8s-context/src/features"
	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/tools/clientcmd"
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

	var getCmd = &cobra.Command{
		Use:   "pods",
		Short: "Get pods in a namespace",
		RunE: func(cmd *cobra.Command, args []string) error {
			interactive, _ := cmd.Flags().GetBool("interactive")
			// filter, _ := cmd.Flags().GetString("filter")

			clientset, err := features.GetClientSet(kubeconfig)
			if err != nil {
				return err
			}

			var ns string
			nsPrompt := promptui.Select{
				Label: "Select namespace",
				Items: []string{"default", "kube-node-lease", "kube-public", "kube-system"},
			}

			if interactive {
				_, ns, err = nsPrompt.Run()
				if err != nil {
					return err
				}
			} else {
				ctx, err := features.GetCurrentContext()
				if err != nil {
					return err
				}
				config, err := clientcmd.LoadFromFile(kubeconfig)
				if err != nil {
					return err
				}
				ns = config.Contexts[ctx].Namespace
			}

			pods, err := clientset.CoreV1().Pods(ns).List(context.Background(), v1.ListOptions{})
			if err != nil {
				return err
			}

			var rows [][]string
			for _, pod := range pods.Items {
				age := duration.HumanDuration(time.Since(pod.CreationTimestamp.Time))
				ownerKind := ""
				ownerName := ""
				for _, owner := range pod.OwnerReferences {
					ownerKind = owner.Kind
					ownerName = owner.Name
					break
				}
				labels := ""
				for k, v := range pod.Labels {
					labels += fmt.Sprintf("%s=%s, ", k, v)
				}
				if len(labels) > 2 {
					labels = labels[:len(labels)-2]
				}
				rows = append(rows, []string{
					pod.Name,
					fmt.Sprintf("%d/%d", pod.Status.ContainerStatuses[0].Ready, len(pod.Spec.Containers)),
					string(pod.Status.Phase),
					strconv.Itoa(int(pod.Status.ContainerStatuses[0].RestartCount)),
					age,
					pod.Spec.Containers[0].Image,
					pod.Spec.NodeName,
					ownerKind,
					ownerName,
					labels,
				})
			}

			headers := []string{"POD NAME", "READY", "STATUS", "RESTARTS", "AGE", "IMAGE", "NODE", "OWNER KIND", "OWNER NAME", "LABELS"}
			table := features.NewSortableTable(headers, rows, 0, "")
			table.Render()

			return nil
		},
	}

	rootCmd := &cobra.Command{Use: "k8s-context"}
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", kubeconfig, "Path to kubeconfig file")

	getCmd.Flags().Bool("interactive", false, "enable interactive mode")
	getCmd.Flags().String("filter", "", "filter by label value")

	rootCmd.AddCommand(versionCmd, loadCmd, mergeCmd, getCmd)

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatalf("error executing command: %v", err)
	}
}
