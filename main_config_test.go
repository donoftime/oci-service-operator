/*
Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func TestLoadControllerManagerConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "controller_manager_config.yaml")

	configBody := `health:
  healthProbeBindAddress: :8082
  readinessEndpointName: ready
  livenessEndpointName: live
metrics:
  bindAddress: 127.0.0.1:9090
  secureServing: true
  certDir: /metrics/certs
  certName: metrics.crt
  keyName: metrics.key
webhook:
  port: 9444
  host: webhook.example
  certDir: /webhook/certs
  certName: webhook.crt
  keyName: webhook.key
leaderElection:
  leaderElect: false
  leaseDuration: 20s
  renewDeadline: 15s
  retryPeriod: 5s
  resourceLock: configmapsleases
  resourceName: custom.oci
  resourceNamespace: custom-ns
cacheNamespace: scoped-ns
syncPeriod: 30s
gracefulShutDown: 45s
controller:
  groupKindConcurrency:
    ReplicaSet.apps: 3
  cacheSyncTimeout: 12s
  recoverPanic: true
`
	assert.NoError(t, os.WriteFile(configPath, []byte(configBody), 0o600))

	config, err := loadControllerManagerConfig(configPath)
	assert.NoError(t, err)
	assert.Equal(t, ":8082", config.Health.HealthProbeBindAddress)
	assert.Equal(t, "ready", config.Health.ReadinessEndpointName)
	assert.Equal(t, "live", config.Health.LivenessEndpointName)
	assert.Equal(t, "127.0.0.1:9090", config.Metrics.BindAddress)
	if assert.NotNil(t, config.Metrics.SecureServing) {
		assert.True(t, *config.Metrics.SecureServing)
	}
	assert.Equal(t, "/metrics/certs", config.Metrics.CertDir)
	assert.Equal(t, "metrics.crt", config.Metrics.CertName)
	assert.Equal(t, "metrics.key", config.Metrics.KeyName)
	if assert.NotNil(t, config.Webhook.Port) {
		assert.Equal(t, 9444, *config.Webhook.Port)
	}
	assert.Equal(t, "webhook.example", config.Webhook.Host)
	assert.Equal(t, "/webhook/certs", config.Webhook.CertDir)
	assert.Equal(t, "webhook.crt", config.Webhook.CertName)
	assert.Equal(t, "webhook.key", config.Webhook.KeyName)
	if assert.NotNil(t, config.LeaderElection) && assert.NotNil(t, config.LeaderElection.LeaderElect) {
		assert.False(t, *config.LeaderElection.LeaderElect)
	}
	assert.Equal(t, "configmapsleases", config.LeaderElection.ResourceLock)
	assert.Equal(t, "custom.oci", config.LeaderElection.ResourceName)
	assert.Equal(t, "custom-ns", config.LeaderElection.ResourceNamespace)
	if assert.NotNil(t, config.LeaderElection.LeaseDuration) {
		assert.Equal(t, 20*time.Second, config.LeaderElection.LeaseDuration.Duration)
	}
	if assert.NotNil(t, config.LeaderElection.RenewDeadline) {
		assert.Equal(t, 15*time.Second, config.LeaderElection.RenewDeadline.Duration)
	}
	if assert.NotNil(t, config.LeaderElection.RetryPeriod) {
		assert.Equal(t, 5*time.Second, config.LeaderElection.RetryPeriod.Duration)
	}
	assert.Equal(t, "scoped-ns", config.CacheNamespace)
	if assert.NotNil(t, config.SyncPeriod) {
		assert.Equal(t, 30*time.Second, config.SyncPeriod.Duration)
	}
	if assert.NotNil(t, config.GracefulShutdownTimeout) {
		assert.Equal(t, 45*time.Second, config.GracefulShutdownTimeout.Duration)
	}
	if assert.NotNil(t, config.Controller) {
		assert.Equal(t, map[string]int{"ReplicaSet.apps": 3}, config.Controller.GroupKindConcurrency)
		if assert.NotNil(t, config.Controller.CacheSyncTimeout) {
			assert.Equal(t, 12*time.Second, config.Controller.CacheSyncTimeout.Duration)
		}
		if assert.NotNil(t, config.Controller.RecoverPanic) {
			assert.True(t, *config.Controller.RecoverPanic)
		}
	}
}

func TestMergeManagerOptionsUsesConfigWhenFlagsAreNotExplicit(t *testing.T) {
	options := defaultManagerOptions(managerFlags{
		metricsAddr:          ":8080",
		probeAddr:            ":8081",
		enableLeaderElection: true,
	})
	leaderElect := false
	config := controllerManagerConfig{}
	config.SyncPeriod = durationPtr(30 * time.Second)
	config.CacheNamespace = "scoped-ns"
	config.GracefulShutdownTimeout = durationPtr(45 * time.Second)
	config.Controller = &controllerManagerController{
		GroupKindConcurrency: map[string]int{"ReplicaSet.apps": 3},
		CacheSyncTimeout:     durationPtr(12 * time.Second),
		RecoverPanic:         boolPtr(true),
	}
	config.Metrics.BindAddress = "127.0.0.1:9090"
	config.Metrics.SecureServing = boolPtr(true)
	config.Metrics.CertDir = "/metrics/certs"
	config.Metrics.CertName = "metrics.crt"
	config.Metrics.KeyName = "metrics.key"
	config.Health.HealthProbeBindAddress = ":8082"
	config.Health.ReadinessEndpointName = "ready"
	config.Health.LivenessEndpointName = "live"
	config.Webhook.Port = intPtr(9444)
	config.Webhook.Host = "webhook.example"
	config.Webhook.CertDir = "/webhook/certs"
	config.Webhook.CertName = "webhook.crt"
	config.Webhook.KeyName = "webhook.key"
	config.LeaderElection = &controllerManagerLeaderElection{
		LeaderElect:       &leaderElect,
		ResourceLock:      "configmapsleases",
		ResourceName:      "custom.oci",
		ResourceNamespace: "custom-ns",
		LeaseDuration:     durationPtr(20 * time.Second),
		RenewDeadline:     durationPtr(15 * time.Second),
		RetryPeriod:       durationPtr(5 * time.Second),
	}

	merged := mergeManagerOptions(options, config, map[string]bool{})
	if assert.NotNil(t, merged.Cache.SyncPeriod) {
		assert.Equal(t, 30*time.Second, *merged.Cache.SyncPeriod)
	}
	assert.Equal(t, map[string]ctrlcache.Config{"scoped-ns": {}}, merged.Cache.DefaultNamespaces)
	assert.Equal(t, "127.0.0.1:9090", merged.Metrics.BindAddress)
	assert.True(t, merged.Metrics.SecureServing)
	assert.Equal(t, "/metrics/certs", merged.Metrics.CertDir)
	assert.Equal(t, "metrics.crt", merged.Metrics.CertName)
	assert.Equal(t, "metrics.key", merged.Metrics.KeyName)
	assert.Equal(t, ":8082", merged.HealthProbeBindAddress)
	assert.Equal(t, "ready", merged.ReadinessEndpointName)
	assert.Equal(t, "live", merged.LivenessEndpointName)
	assert.False(t, merged.LeaderElection)
	assert.Equal(t, "configmapsleases", merged.LeaderElectionResourceLock)
	assert.Equal(t, "custom.oci", merged.LeaderElectionID)
	assert.Equal(t, "custom-ns", merged.LeaderElectionNamespace)
	if assert.NotNil(t, merged.LeaseDuration) {
		assert.Equal(t, 20*time.Second, *merged.LeaseDuration)
	}
	if assert.NotNil(t, merged.RenewDeadline) {
		assert.Equal(t, 15*time.Second, *merged.RenewDeadline)
	}
	if assert.NotNil(t, merged.RetryPeriod) {
		assert.Equal(t, 5*time.Second, *merged.RetryPeriod)
	}
	if assert.NotNil(t, merged.GracefulShutdownTimeout) {
		assert.Equal(t, 45*time.Second, *merged.GracefulShutdownTimeout)
	}
	assert.Equal(t, map[string]int{"ReplicaSet.apps": 3}, merged.Controller.GroupKindConcurrency)
	assert.Equal(t, 12*time.Second, merged.Controller.CacheSyncTimeout)
	if assert.NotNil(t, merged.Controller.RecoverPanic) {
		assert.True(t, *merged.Controller.RecoverPanic)
	}
	if assert.IsType(t, &webhook.DefaultServer{}, merged.WebhookServer) {
		server := merged.WebhookServer.(*webhook.DefaultServer)
		assert.Equal(t, 9444, server.Options.Port)
		assert.Equal(t, "webhook.example", server.Options.Host)
		assert.Equal(t, "/webhook/certs", server.Options.CertDir)
		assert.Equal(t, "webhook.crt", server.Options.CertName)
		assert.Equal(t, "webhook.key", server.Options.KeyName)
	}
}

func TestMergeManagerOptionsPrefersExplicitFlags(t *testing.T) {
	options := defaultManagerOptions(managerFlags{
		metricsAddr:          ":8080",
		probeAddr:            ":8081",
		enableLeaderElection: true,
	})

	leaderElect := false
	config := controllerManagerConfig{}
	config.Metrics.BindAddress = "127.0.0.1:9090"
	config.Metrics.SecureServing = boolPtr(true)
	config.Health.HealthProbeBindAddress = ":8082"
	config.LeaderElection = &controllerManagerLeaderElection{LeaderElect: &leaderElect}

	merged := mergeManagerOptions(options, config, map[string]bool{
		"metrics-bind-address":      true,
		"health-probe-bind-address": true,
		"leader-elect":              true,
	})
	assert.Equal(t, ":8080", merged.Metrics.BindAddress)
	assert.True(t, merged.Metrics.SecureServing)
	assert.Equal(t, ":8081", merged.HealthProbeBindAddress)
	assert.True(t, merged.LeaderElection)
}

func TestMergeManagerOptionsDoesNotOverrideExistingNonFlagOptions(t *testing.T) {
	options := defaultManagerOptions(managerFlags{
		metricsAddr:          ":8080",
		probeAddr:            ":8081",
		enableLeaderElection: true,
	})
	existingNamespace := "existing"
	options.Cache.DefaultNamespaces = map[string]ctrlcache.Config{existingNamespace: {}}
	options.Controller = config.Controller{CacheSyncTimeout: 3 * time.Second}
	options.WebhookServer = webhook.NewServer(webhook.Options{Port: 9443, Host: "existing-host"})

	configFile := controllerManagerConfig{
		CacheNamespace: "from-config",
		Controller: &controllerManagerController{
			GroupKindConcurrency: map[string]int{"ReplicaSet.apps": 3},
			CacheSyncTimeout:     durationPtr(12 * time.Second),
		},
		Webhook: controllerManagerWebhook{Port: intPtr(9444), Host: "webhook.example"},
	}

	merged := mergeManagerOptions(options, configFile, map[string]bool{})
	assert.Equal(t, map[string]ctrlcache.Config{existingNamespace: {}}, merged.Cache.DefaultNamespaces)
	assert.Equal(t, 3*time.Second, merged.Controller.CacheSyncTimeout)
	assert.IsType(t, &webhook.DefaultServer{}, merged.WebhookServer)
	server := merged.WebhookServer.(*webhook.DefaultServer)
	assert.Equal(t, 9443, server.Options.Port)
	assert.Equal(t, "existing-host", server.Options.Host)
	assert.Equal(t, map[string]int{"ReplicaSet.apps": 3}, merged.Controller.GroupKindConcurrency)
}

func durationPtr(value time.Duration) *controllerManagerDuration {
	return &controllerManagerDuration{Duration: value}
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}
