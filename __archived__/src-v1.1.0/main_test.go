package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopscorner/k8s-context/src/features"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

var (
	tempFile string
	log      = logrus.New()
)

func TestLoad(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	t.Run("SingleFile", func(t *testing.T) {
		tempFile, err := ioutil.TempFile("", "test-config-*.yaml")
		if err != nil {
			log.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tempFile.Name())

		testConfig := &api.Config{
			AuthInfos: map[string]*api.AuthInfo{
				"test-user": {
					Username: "test-username",
					Password: "test-password",
				},
			},
			Clusters: map[string]*api.Cluster{
				"test-cluster": {
					Server: "https://test-server:1234",
				},
			},
			Contexts: map[string]*api.Context{
				"test-context": {
					AuthInfo:  "test-user",
					Cluster:   "test-cluster",
					Namespace: "test-namespace",
				},
			},
			CurrentContext: "test-context",
			Kind:           "Config",
			APIVersion:     "v1",
		}
		if err := clientcmd.WriteToFile(*testConfig, tempFile.Name()); err != nil {
			log.Fatalf("failed to write test config: %v", err)
		}

		kc := &features.KubeConfig{
			Files: []string{tempFile.Name()},
		}
		err = kc.Load()
		assert.NoError(t, err)
		assert.NotNil(t, kc.Merged)

		merged := kc.Merged
		assert.Equal(t, testConfig.AuthInfos, merged.AuthInfos)
		assert.Equal(t, testConfig.Clusters, merged.Clusters)
		assert.Equal(t, testConfig.Contexts, merged.Contexts)
		assert.Equal(t, testConfig.CurrentContext, merged.CurrentContext)
		assert.Equal(t, testConfig.Kind, merged.Kind)
		assert.Equal(t, testConfig.APIVersion, merged.APIVersion)
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		tempDir, err := ioutil.TempDir("", "test-configs-*")
		if err != nil {
			log.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		testConfigs := []*api.Config{
			{
				AuthInfos: map[string]*api.AuthInfo{
					"test-user1": {
						Username: "test-username1",
						Password: "test-password1",
					},
				},
				Clusters: map[string]*api.Cluster{
					"test-cluster1": {
						Server: "https://test-server1:1234",
					},
				},
				Contexts: map[string]*api.Context{
					"test-context1": {
						AuthInfo:  "test-user1",
						Cluster:   "test-cluster1",
						Namespace: "test-namespace1",
					},
				},
				CurrentContext: "test-context1",
				Kind:           "Config",
				APIVersion:     "v1",
			},
			{
				AuthInfos: map[string]*api.AuthInfo{
					"test-user2": {
						Username: "test-username2",
						Password: "test-password2",
					},
				},
				Clusters: map[string]*api.Cluster{
					"test-cluster2": {
						Server: "https://test-server2:1234",
					},
				},
				Contexts: map[string]*api.Context{
					"test-context2": {
						AuthInfo:  "test-user2",
						Cluster:   "test-cluster2",
						Namespace: "test-namespace2",
					},
				},
				CurrentContext: "test-context2",
				Kind:           "Config",
				APIVersion:     "v1",
			},
		}
		for i, testConfig := range testConfigs {
			tempFile := filepath.Join(tempDir, fmt.Sprintf("config%d.yaml", i))
			if err := clientcmd.WriteToFile(*testConfig, tempFile); err != nil {
				log.Fatalf("failed to write test config: %v", err)
			}
		}

		kc := &features.KubeConfig{
			Files: []string{tempDir},
		}
		err = kc.Load()
		assert.NoError(t, err)
		assert.NotNil(t, kc.Merged)

		merged := kc.Merged
		assert.Equal(t, testConfigs[0].AuthInfos, merged.AuthInfos)
		assert.Equal(t, testConfigs[0].Clusters, merged.Clusters)
		assert.Equal(t, testConfigs[0].Contexts, merged.Contexts)
		assert.Equal(t, testConfigs[0].CurrentContext, merged.CurrentContext)
		assert.Equal(t, testConfigs[0].Kind, merged.Kind)
		assert.Equal(t, testConfigs[0].APIVersion, merged.APIVersion)

		for _, testConfig := range testConfigs[1:] {
			assert.Equal(t, testConfig.AuthInfos, merged.AuthInfos)
			assert.Equal(t, testConfig.Clusters, merged.Clusters)
			assert.Equal(t, testConfig.Contexts, merged.Contexts)
			assert.Equal(t, testConfig.CurrentContext, merged.CurrentContext)
		}
	})
}

func TestSaveToFile(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	tempFile, err := ioutil.TempFile("", "test-config-*.yaml")
	if err != nil {
		log.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	kc := &features.KubeConfig{
		Merged: &api.Config{
			AuthInfos: map[string]*api.AuthInfo{
				"test-user": {
					Username: "test-username",
					Password: "test-password",
				},
			},
			Clusters: map[string]*api.Cluster{
				"test-cluster": {
					Server: "https://test-server:1234",
				},
			},
			Contexts: map[string]*api.Context{
				"test-context": {
					AuthInfo:  "test-user",
					Cluster:   "test-cluster",
					Namespace: "test-namespace",
				},
			},
			CurrentContext: "test-context",
			Kind:           "Config",
			APIVersion:     "v1",
		},
	}

	err = kc.SaveToFile(tempFile.Name())
	assert.NoError(t, err)
	loaded, err := clientcmd.LoadFromFile(tempFile.Name())
	assert.NoError(t, err)

	assert.Equal(t, kc.Merged.AuthInfos, loaded.AuthInfos)
	assert.Equal(t, kc.Merged.Clusters, loaded.Clusters)
	assert.Equal(t, kc.Merged.Contexts, loaded.Contexts)
	assert.Equal(t, kc.Merged.CurrentContext, loaded.CurrentContext)
	assert.Equal(t, kc.Merged.Kind, loaded.Kind)
	assert.Equal(t, kc.Merged.APIVersion, loaded.APIVersion)
}

func TestGetPods(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stdout)

	kc := &features.KubeConfig{
		Merged: &api.Config{
			AuthInfos: map[string]*api.AuthInfo{
				"test-user": {
					Username: "test-username",
					Password: "test-password",
				},
			},
			Clusters: map[string]*api.Cluster{
				"test-cluster": {
					Server: "https://test-server:1234",
				},
			},
			Contexts: map[string]*api.Context{
				"test-context": {
					AuthInfo:  "test-user",
					Cluster:   "test-cluster",
					Namespace: "test-namespace",
				},
			},
			CurrentContext: "test-context",
			Kind:           "Config",
			APIVersion:     "v1",
		},
	}

	pods, err := kc.GetPods()
	assert.NoError(t, err)
	assert.NotNil(t, pods)
}
