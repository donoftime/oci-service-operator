/*
Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"fmt"
	"os"

	"github.com/oracle/oci-go-sdk/v65/common"

	"github.com/oracle/oci-service-operator/go_ensurefips"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	ociv1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup")}
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(ociv1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	if err := run(); err != nil {
		setupLog.ErrorLog(err, "manager startup failed")
		os.Exit(1)
	}
}

func run() error {
	go_ensurefips.Compliant()
	common.EnableInstanceMetadataServiceLookup()

	flags, zapOptions, explicitFlags := parseManagerFlags()
	ctrl.SetLogger(newZapLogger(zapOptions))

	managerOptions, err := buildManagerOptions(flags, explicitFlags)
	if err != nil {
		return fmt.Errorf("build manager options: %w", err)
	}

	manager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), managerOptions)
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	initializeOSOKResources(flags.initOSOKResources, manager)

	provider, metricsClient, credClient, err := buildRuntimeDependencies(manager)
	if err != nil {
		return err
	}

	if err := registerControllers(manager, provider, credClient, metricsClient); err != nil {
		return err
	}
	if err := registerHealthChecks(manager); err != nil {
		return err
	}

	setupLog.InfoLog("starting manager")
	if err := manager.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("start manager: %w", err)
	}

	return nil
}
