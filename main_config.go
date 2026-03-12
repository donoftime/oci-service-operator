/*
Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"gopkg.in/yaml.v3"
)

const defaultLeaderElectionID = "40558063.oci"

type managerFlags struct {
	configFile           string
	metricsAddr          string
	probeAddr            string
	enableLeaderElection bool
	initOSOKResources    bool
}

type controllerManagerConfig struct {
	SyncPeriod              *controllerManagerDuration       `yaml:"syncPeriod,omitempty"`
	CacheNamespace          string                           `yaml:"cacheNamespace,omitempty"`
	GracefulShutdownTimeout *controllerManagerDuration       `yaml:"gracefulShutDown,omitempty"`
	Controller              *controllerManagerController     `yaml:"controller,omitempty"`
	Metrics                 controllerManagerMetrics         `yaml:"metrics,omitempty"`
	Health                  controllerManagerHealth          `yaml:"health,omitempty"`
	Webhook                 controllerManagerWebhook         `yaml:"webhook,omitempty"`
	LeaderElection          *controllerManagerLeaderElection `yaml:"leaderElection,omitempty"`
}

type controllerManagerController struct {
	GroupKindConcurrency map[string]int             `yaml:"groupKindConcurrency,omitempty"`
	CacheSyncTimeout     *controllerManagerDuration `yaml:"cacheSyncTimeout,omitempty"`
	RecoverPanic         *bool                      `yaml:"recoverPanic,omitempty"`
}

type controllerManagerMetrics struct {
	BindAddress   string `yaml:"bindAddress,omitempty"`
	SecureServing *bool  `yaml:"secureServing,omitempty"`
	CertDir       string `yaml:"certDir,omitempty"`
	CertName      string `yaml:"certName,omitempty"`
	KeyName       string `yaml:"keyName,omitempty"`
}

type controllerManagerHealth struct {
	HealthProbeBindAddress string `yaml:"healthProbeBindAddress,omitempty"`
	ReadinessEndpointName  string `yaml:"readinessEndpointName,omitempty"`
	LivenessEndpointName   string `yaml:"livenessEndpointName,omitempty"`
}

type controllerManagerWebhook struct {
	Port     *int   `yaml:"port,omitempty"`
	Host     string `yaml:"host,omitempty"`
	CertDir  string `yaml:"certDir,omitempty"`
	CertName string `yaml:"certName,omitempty"`
	KeyName  string `yaml:"keyName,omitempty"`
}

type controllerManagerLeaderElection struct {
	LeaderElect       *bool                      `yaml:"leaderElect,omitempty"`
	LeaseDuration     *controllerManagerDuration `yaml:"leaseDuration,omitempty"`
	RenewDeadline     *controllerManagerDuration `yaml:"renewDeadline,omitempty"`
	RetryPeriod       *controllerManagerDuration `yaml:"retryPeriod,omitempty"`
	ResourceLock      string                     `yaml:"resourceLock,omitempty"`
	ResourceName      string                     `yaml:"resourceName,omitempty"`
	ResourceNamespace string                     `yaml:"resourceNamespace,omitempty"`
}

type controllerManagerDuration struct {
	time.Duration
}

func (d *controllerManagerDuration) UnmarshalYAML(node *yaml.Node) error {
	if node == nil || node.Tag == "!!null" {
		return nil
	}

	var raw string
	if err := node.Decode(&raw); err == nil {
		duration, parseErr := time.ParseDuration(raw)
		if parseErr != nil {
			return fmt.Errorf("parse duration %q: %w", raw, parseErr)
		}
		d.Duration = duration
		return nil
	}

	var rawInt int64
	if err := node.Decode(&rawInt); err == nil {
		d.Duration = time.Duration(rawInt)
		return nil
	}

	return fmt.Errorf("unsupported duration value %q", node.Value)
}

func parseManagerFlags() (managerFlags, zap.Options, map[string]bool) {
	flags := managerFlags{}
	zapOptions := zap.Options{Development: true}

	flag.StringVar(&flags.configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	flag.StringVar(&flags.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&flags.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&flags.enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&flags.initOSOKResources, "init-osok-resources", false,
		"Install OSOK prerequisites like CRDs and Webhooks at manager bootup")

	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()

	explicitFlags := map[string]bool{}
	flag.CommandLine.Visit(func(setFlag *flag.Flag) {
		explicitFlags[setFlag.Name] = true
	})

	return flags, zapOptions, explicitFlags
}

func newZapLogger(options zap.Options) logr.Logger {
	return zap.New(zap.UseFlagOptions(&options))
}

func buildManagerOptions(flags managerFlags, explicitFlags map[string]bool) (ctrl.Options, error) {
	options := defaultManagerOptions(flags)
	if flags.configFile == "" {
		setupLog.InfoLog("Loading the configuration from the command arguments")
		return options, nil
	}

	setupLog.InfoLog("Loading the configuration from the ControllerManagerConfig configMap")
	config, err := loadControllerManagerConfig(flags.configFile)
	if err != nil {
		return ctrl.Options{}, err
	}

	return mergeManagerOptions(options, config, explicitFlags), nil
}

func defaultManagerOptions(flags managerFlags) ctrl.Options {
	return ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: flags.metricsAddr},
		HealthProbeBindAddress: flags.probeAddr,
		LeaderElection:         flags.enableLeaderElection,
		LeaderElectionID:       defaultLeaderElectionID,
	}
}

func loadControllerManagerConfig(path string) (controllerManagerConfig, error) {
	var config controllerManagerConfig

	data, err := os.ReadFile(path)
	if err != nil {
		return controllerManagerConfig{}, err
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return controllerManagerConfig{}, err
	}

	return config, nil
}

func mergeManagerOptions(options ctrl.Options, config controllerManagerConfig, explicitFlags map[string]bool) ctrl.Options {
	applyCacheOptions(&options, config)
	applyMetricsOptions(&options, config, explicitFlags)
	applyHealthOptions(&options, config, explicitFlags)
	applyLeaderElectionOptions(&options, config, explicitFlags)
	applyShutdownOptions(&options, config)
	applyControllerOptions(&options, config)
	applyWebhookOptions(&options, config)

	return options
}

func applyCacheOptions(options *ctrl.Options, config controllerManagerConfig) {
	if options.Cache.SyncPeriod == nil && config.SyncPeriod != nil {
		options.Cache.SyncPeriod = &config.SyncPeriod.Duration
	}
	if len(options.Cache.DefaultNamespaces) == 0 && config.CacheNamespace != "" {
		options.Cache.DefaultNamespaces = map[string]ctrlcache.Config{config.CacheNamespace: {}}
	}
}

func applyMetricsOptions(options *ctrl.Options, config controllerManagerConfig, explicitFlags map[string]bool) {
	if !explicitFlags["metrics-bind-address"] && config.Metrics.BindAddress != "" {
		options.Metrics.BindAddress = config.Metrics.BindAddress
	}
	if config.Metrics.SecureServing != nil {
		options.Metrics.SecureServing = *config.Metrics.SecureServing
	}
	if options.Metrics.CertDir == "" && config.Metrics.CertDir != "" {
		options.Metrics.CertDir = config.Metrics.CertDir
	}
	if options.Metrics.CertName == "" && config.Metrics.CertName != "" {
		options.Metrics.CertName = config.Metrics.CertName
	}
	if options.Metrics.KeyName == "" && config.Metrics.KeyName != "" {
		options.Metrics.KeyName = config.Metrics.KeyName
	}
}

func applyHealthOptions(options *ctrl.Options, config controllerManagerConfig, explicitFlags map[string]bool) {
	if !explicitFlags["health-probe-bind-address"] && config.Health.HealthProbeBindAddress != "" {
		options.HealthProbeBindAddress = config.Health.HealthProbeBindAddress
	}
	if options.ReadinessEndpointName == "" && config.Health.ReadinessEndpointName != "" {
		options.ReadinessEndpointName = config.Health.ReadinessEndpointName
	}
	if options.LivenessEndpointName == "" && config.Health.LivenessEndpointName != "" {
		options.LivenessEndpointName = config.Health.LivenessEndpointName
	}
}

func applyLeaderElectionOptions(options *ctrl.Options, config controllerManagerConfig, explicitFlags map[string]bool) {
	if config.LeaderElection == nil {
		return
	}

	applyLeaderElectionCoreOptions(options, *config.LeaderElection, explicitFlags)
	applyLeaderElectionDurations(options, *config.LeaderElection)
}

func applyLeaderElectionCoreOptions(options *ctrl.Options, config controllerManagerLeaderElection, explicitFlags map[string]bool) {
	if !explicitFlags["leader-elect"] && config.LeaderElect != nil {
		options.LeaderElection = *config.LeaderElect
	}
	if options.LeaderElectionResourceLock == "" && config.ResourceLock != "" {
		options.LeaderElectionResourceLock = config.ResourceLock
	}
	if options.LeaderElectionNamespace == "" && config.ResourceNamespace != "" {
		options.LeaderElectionNamespace = config.ResourceNamespace
	}
	if config.ResourceName != "" {
		options.LeaderElectionID = config.ResourceName
	}
}

func applyLeaderElectionDurations(options *ctrl.Options, config controllerManagerLeaderElection) {
	if options.LeaseDuration == nil && config.LeaseDuration != nil {
		options.LeaseDuration = &config.LeaseDuration.Duration
	}
	if options.RenewDeadline == nil && config.RenewDeadline != nil {
		options.RenewDeadline = &config.RenewDeadline.Duration
	}
	if options.RetryPeriod == nil && config.RetryPeriod != nil {
		options.RetryPeriod = &config.RetryPeriod.Duration
	}
}

func applyShutdownOptions(options *ctrl.Options, config controllerManagerConfig) {
	if options.GracefulShutdownTimeout == nil && config.GracefulShutdownTimeout != nil {
		options.GracefulShutdownTimeout = &config.GracefulShutdownTimeout.Duration
	}
}

func applyControllerOptions(options *ctrl.Options, config controllerManagerConfig) {
	if config.Controller == nil {
		return
	}

	if options.Controller.CacheSyncTimeout == 0 && config.Controller.CacheSyncTimeout != nil {
		options.Controller.CacheSyncTimeout = config.Controller.CacheSyncTimeout.Duration
	}
	if len(options.Controller.GroupKindConcurrency) == 0 && len(config.Controller.GroupKindConcurrency) > 0 {
		options.Controller.GroupKindConcurrency = config.Controller.GroupKindConcurrency
	}
	if options.Controller.RecoverPanic == nil && config.Controller.RecoverPanic != nil {
		options.Controller.RecoverPanic = config.Controller.RecoverPanic
	}
}

func applyWebhookOptions(options *ctrl.Options, config controllerManagerConfig) {
	if options.WebhookServer == nil && shouldConfigureWebhookServer(config.Webhook) {
		options.WebhookServer = webhook.NewServer(webhook.Options{
			Port:     webhookPort(config.Webhook.Port),
			Host:     config.Webhook.Host,
			CertDir:  config.Webhook.CertDir,
			CertName: config.Webhook.CertName,
			KeyName:  config.Webhook.KeyName,
		})
	}
}

func shouldConfigureWebhookServer(config controllerManagerWebhook) bool {
	return config.Port != nil || config.Host != "" || config.CertDir != "" || config.CertName != "" || config.KeyName != ""
}

func webhookPort(port *int) int {
	if port == nil {
		return 0
	}
	return *port
}
