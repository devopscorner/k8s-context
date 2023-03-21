package main

import (
	"errors"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var rootCmd = &cobra.Command{
	Use:   "k8s-context",
	Short: "A tool to change the current context in a Kubernetes configuration file",
	RunE:  changeContextCmd,
}

var (
	kubeconfigPath string
	contextName    string
)

func init() {
	// If the user didn't specify a kubeconfig file, use the default location.
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
		}
	}

	if contextName == "" {
		contextName = "devopscorner-context"
	}

	rootCmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to the Kubernetes configuration file")
	rootCmd.MarkFlagRequired("kubeconfig")

	rootCmd.Flags().StringVar(&contextName, "context", "", "Context name (kubernetes cluster)")
	rootCmd.MarkFlagRequired("context")

	// Configure Logrus to output JSON to stdout.
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		// logMessage("Failed to execute command", err, log.ErrorLevel)
		// fmt.Fprintf(os.Stderr, "Failed to execute command '%s'", err)
		os.Exit(1)
	}
}

func changeContextCmd(cmd *cobra.Command, args []string) error {
	// Change the current context.
	if err := ChangeKubeconfigContext(kubeconfigPath, contextName); err != nil {
		// logMessage(" Failed to change Kubernetes context "+contextName, err, log.ErrorLevel)
		return err
	}

	// Log success.
	logMessage("Successfully changed context to "+contextName, nil, log.InfoLevel)
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

	return nil
}

func logMessage(message string, err error, level log.Level) {
	if err != nil {
		log.WithError(err).Log(level, message)
	} else {
		log.Info(level, message)
	}
}
