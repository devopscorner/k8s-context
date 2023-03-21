package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
)

func TestChangeKubeconfigContext(t *testing.T) {
	// Redirect logrus output to a buffer for testing.
	var logOutput bytes.Buffer
	log.SetOutput(&logOutput)

	// Create a temporary test directory.
	tmpdir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	// Create a test configuration file.
	kubeconfigPath := tmpdir + "/kubeconfig"
	kubeconfigContents := `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: REDACTED
    server: https://localhost:6443
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: admin
  name: kubernetes-admin@kubernetes
- context:
    cluster: devopscorner-cluster
    user: admin
  name: devopscorner-cluster
current-context: kubernetes-admin@kubernetes
kind: Config
preferences: {}
users:
- name: admin
  user:
    client-certificate-data: REDACTED
    client-key-data: REDACTED
`
	err = ioutil.WriteFile(kubeconfigPath, []byte(kubeconfigContents), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Change the context in the test configuration file.
	err = ChangeKubeconfigContext(kubeconfigPath, "devopscorner-cluster")
	if err != nil {
		t.Fatal(err)
	}

	// Verify that the context was changed correctly.
	kubeconfigBytes, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		t.Fatal(err)
	}
	kubeconfig, err := clientcmd.Load(kubeconfigBytes)
	if err != nil {
		t.Fatal(err)
	}
	if kubeconfig.CurrentContext != "devopscorner-cluster" {
		t.Errorf("expected current context to be 'devopscorner-cluster', got '%s'", kubeconfig.CurrentContext)
	}

	// Verify logs.
	expectedLogs := []struct {
		level log.Level
		msg   string
	}{
		{level: log.InfoLevel, msg: "Current context: kubernetes-admin@kubernetes"},
		{level: log.InfoLevel, msg: "Successfully changed context to 'devopscorner-cluster'"},
	}
	logLines := logOutput.String()
	for _, expectedLog := range expectedLogs {
		if !assertLogLine(t, logLines, expectedLog.level, expectedLog.msg) {
			return
		}
	}
}

func assertLogLine(t *testing.T, logs string, level log.Level, message string) bool {
	if !assert.Contains(t, logs, message) {
		return false
	}

	expected := fmt.Sprintf(`"level":"%s"`, level)
	if !assert.Contains(t, logs, expected) {
		return false
	}

	return true
}
