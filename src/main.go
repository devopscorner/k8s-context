package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	Log            *log.Logger
	kubeconfigPath string
	contextName    string
)

func init() {
	// Configure Logrus to output JSON to stdout.
	// log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
}

func main() {
	// Define a command-line flag for the Kubernetes configuration file.
	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to the Kubernetes configuration file")
	flag.StringVar(&contextName, "context", "", "Context name (kubernetes cluster)")

	// Parse the command-line flags.
	flag.Parse()

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

	// Change the current context.
	if err := ChangeKubeconfigContext(kubeconfigPath, contextName); err != nil {
		logMessage(" Failed to change Kubernetes context "+contextName, err, log.ErrorLevel)
		return
	}

	// Log success.
	logMessage(" Successfully changed context to "+contextName, nil, log.InfoLevel)
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
		return errors.New(" context does not exist in the Kubernetes configuration file")
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
