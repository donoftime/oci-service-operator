/*
Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/oracle/oci-service-operator/controllers"
	"github.com/oracle/oci-service-operator/pkg/authhelper"
	"github.com/oracle/oci-service-operator/pkg/config"
	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/credhelper/kubesecret"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	ociapigw "github.com/oracle/oci-service-operator/pkg/servicemanager/apigateway"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/autonomousdatabases/adb"
	ocicompute "github.com/oracle/oci-service-operator/pkg/servicemanager/compute"
	ocicontainerinstance "github.com/oracle/oci-service-operator/pkg/servicemanager/containerinstance"
	ocidataflow "github.com/oracle/oci-service-operator/pkg/servicemanager/dataflow"
	ocifunctions "github.com/oracle/oci-service-operator/pkg/servicemanager/functions"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/mysql/dbsystem"
	ocinetworking "github.com/oracle/oci-service-operator/pkg/servicemanager/networking"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/nosql"
	ociobjectstorage "github.com/oracle/oci-service-operator/pkg/servicemanager/objectstorage"
	opensearchmanager "github.com/oracle/oci-service-operator/pkg/servicemanager/opensearch"
	ocipostgres "github.com/oracle/oci-service-operator/pkg/servicemanager/postgresql"
	ociqueue "github.com/oracle/oci-service-operator/pkg/servicemanager/queue"
	ociredis "github.com/oracle/oci-service-operator/pkg/servicemanager/redis"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/streams"
	"github.com/oracle/oci-service-operator/pkg/util"
)

type controllerRegistration struct {
	name  string
	setup func() error
}

func initializeOSOKResources(initOSOKResources bool, manager ctrl.Manager) {
	if !initOSOKResources {
		return
	}

	util.InitOSOK(manager.GetConfig(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("initOSOK")})
}

func buildRuntimeDependencies(manager ctrl.Manager) (common.ConfigurationProvider, *metrics.Metrics, credhelper.CredentialClient, error) {
	setupLog.InfoLog("Getting the config details")
	osokConfig := config.GetConfigDetails(loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config")})

	authConfigProvider := &authhelper.AuthConfigProvider{
		Log: loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config")},
	}

	provider, err := authConfigProvider.GetAuthProvider(osokConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get oci configuration provider: %w", err)
	}

	metricsClient := metrics.Init("osok", loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("metrics")})
	credentialClient := &kubesecret.KubeSecretClient{
		Client:  manager.GetClient(),
		Log:     loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("credential-helper").WithName("KubeSecretClient")},
		Metrics: metricsClient,
	}

	return provider, metricsClient, credentialClient, nil
}

func registerControllers(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	for _, registration := range controllerRegistrations(manager, provider, credentialClient, metricsClient) {
		if err := registration.setup(); err != nil {
			return fmt.Errorf("setup %s controller: %w", registration.name, err)
		}
	}

	return nil
}

func controllerRegistrations(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) []controllerRegistration {
	return []controllerRegistration{
		{name: "AutonomousDatabases", setup: func() error {
			return setupAutonomousDatabasesController(manager, provider, credentialClient, metricsClient)
		}},
		{name: "Streams", setup: func() error { return setupStreamsController(manager, provider, credentialClient, metricsClient) }},
		{name: "MySqlDbSystem", setup: func() error { return setupMySQLDBSystemController(manager, provider, credentialClient, metricsClient) }},
		{name: "RedisCluster", setup: func() error { return setupRedisClusterController(manager, provider, credentialClient, metricsClient) }},
		{name: "PostgresDbSystem", setup: func() error {
			return setupPostgresDBSystemController(manager, provider, credentialClient, metricsClient)
		}},
		{name: "ApiGateway", setup: func() error { return setupAPIGatewayController(manager, provider, credentialClient, metricsClient) }},
		{name: "ApiGatewayDeployment", setup: func() error {
			return setupAPIGatewayDeploymentController(manager, provider, credentialClient, metricsClient)
		}},
		{name: "NoSQLDatabase", setup: func() error { return setupNoSQLDatabaseController(manager, provider, credentialClient, metricsClient) }},
		{name: "OpenSearchCluster", setup: func() error {
			return setupOpenSearchClusterController(manager, provider, credentialClient, metricsClient)
		}},
		{name: "OciQueue", setup: func() error { return setupQueueController(manager, provider, credentialClient, metricsClient) }},
		{name: "ObjectStorageBucket", setup: func() error { return setupObjectStorageController(manager, provider, credentialClient, metricsClient) }},
		{name: "FunctionsApplication", setup: func() error {
			return setupFunctionsApplicationController(manager, provider, credentialClient, metricsClient)
		}},
		{name: "FunctionsFunction", setup: func() error {
			return setupFunctionsFunctionController(manager, provider, credentialClient, metricsClient)
		}},
		{name: "DataFlowApplication", setup: func() error { return setupDataFlowController(manager, provider, credentialClient, metricsClient) }},
		{name: "ContainerInstance", setup: func() error {
			return setupContainerInstanceController(manager, provider, credentialClient, metricsClient)
		}},
		{name: "ComputeInstance", setup: func() error {
			return setupComputeInstanceController(manager, provider, credentialClient, metricsClient)
		}},
		{name: "OciVcn", setup: func() error { return setupVCNController(manager, provider, credentialClient, metricsClient) }},
		{name: "OciSubnet", setup: func() error { return setupSubnetController(manager, provider, credentialClient, metricsClient) }},
		{name: "OciInternetGateway", setup: func() error {
			return setupInternetGatewayController(manager, provider, credentialClient, metricsClient)
		}},
		{name: "OciNatGateway", setup: func() error { return setupNatGatewayController(manager, provider, credentialClient, metricsClient) }},
		{name: "OciServiceGateway", setup: func() error { return setupServiceGatewayController(manager, provider, credentialClient, metricsClient) }},
		{name: "OciDrg", setup: func() error { return setupDRGController(manager, provider, credentialClient, metricsClient) }},
		{name: "OciSecurityList", setup: func() error { return setupSecurityListController(manager, provider, credentialClient, metricsClient) }},
		{name: "OciNetworkSecurityGroup", setup: func() error {
			return setupNetworkSecurityGroupController(manager, provider, credentialClient, metricsClient)
		}},
		{name: "OciRouteTable", setup: func() error { return setupRouteTableController(manager, provider, credentialClient, metricsClient) }},
	}
}

func registerHealthChecks(manager ctrl.Manager) error {
	if err := manager.AddHealthzCheck("health", healthz.Ping); err != nil {
		return fmt.Errorf("set up health check: %w", err)
	}
	if err := manager.AddReadyzCheck("check", healthz.Ping); err != nil {
		return fmt.Errorf("set up ready check: %w", err)
	}

	return nil
}

func newBaseReconciler(manager ctrl.Manager, serviceManager servicemanager.OSOKServiceManager, controllerName string, metricsClient *metrics.Metrics) *core.BaseReconciler {
	return &core.BaseReconciler{
		Client:             manager.GetClient(),
		OSOKServiceManager: serviceManager,
		Finalizer:          core.NewBaseFinalizer(manager.GetClient(), ctrl.Log),
		Log:                controllerLogger(controllerName),
		Metrics:            metricsClient,
		Recorder:           manager.GetEventRecorderFor(controllerName),
		Scheme:             scheme,
	}
}

func controllerLogger(name string) loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName(name)}
}

func serviceManagerLogger(name string) loggerutil.OSOKLogger {
	return loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName(name)}
}

func setupAutonomousDatabasesController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.AutonomousDatabasesReconciler{
		Reconciler: newBaseReconciler(manager, adb.NewAdbServiceManager(provider, credentialClient, scheme, serviceManagerLogger("AutonomousDatabases")), "AutonomousDatabases", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupStreamsController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.StreamReconciler{
		Reconciler: newBaseReconciler(manager, streams.NewStreamServiceManager(provider, credentialClient, scheme, serviceManagerLogger("Streams"), metricsClient), "Streams", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupMySQLDBSystemController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.MySqlDBsystemReconciler{
		Reconciler: newBaseReconciler(manager, dbsystem.NewDbSystemServiceManager(provider, credentialClient, scheme, serviceManagerLogger("MySqlDbSystem")), "MySqlDbSystem", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupRedisClusterController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.RedisClusterReconciler{
		Reconciler: newBaseReconciler(manager, ociredis.NewRedisClusterServiceManager(provider, credentialClient, scheme, serviceManagerLogger("RedisCluster")), "RedisCluster", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupPostgresDBSystemController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.PostgresDbSystemReconciler{
		Reconciler: newBaseReconciler(manager, ocipostgres.NewPostgresDbSystemServiceManager(provider, credentialClient, scheme, serviceManagerLogger("PostgresDbSystem")), "PostgresDbSystem", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupAPIGatewayController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.ApiGatewayReconciler{
		Reconciler: newBaseReconciler(manager, ociapigw.NewGatewayServiceManager(provider, credentialClient, scheme, serviceManagerLogger("ApiGateway")), "ApiGateway", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupAPIGatewayDeploymentController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.ApiGatewayDeploymentReconciler{
		Reconciler: newBaseReconciler(manager, ociapigw.NewDeploymentServiceManager(provider, credentialClient, scheme, serviceManagerLogger("ApiGatewayDeployment")), "ApiGatewayDeployment", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupNoSQLDatabaseController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.NoSQLDatabaseReconciler{
		Reconciler: newBaseReconciler(manager, nosql.NewNoSQLDatabaseServiceManager(provider, credentialClient, scheme, serviceManagerLogger("NoSQLDatabase")), "NoSQLDatabase", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupOpenSearchClusterController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.OpenSearchClusterReconciler{
		Reconciler: newBaseReconciler(manager, opensearchmanager.NewOpenSearchClusterServiceManager(provider, credentialClient, scheme, serviceManagerLogger("OpenSearchCluster"), metricsClient), "OpenSearchCluster", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupQueueController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.OciQueueReconciler{
		Reconciler: newBaseReconciler(manager, ociqueue.NewOciQueueServiceManager(provider, credentialClient, scheme, serviceManagerLogger("OciQueue")), "OciQueue", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupObjectStorageController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.ObjectStorageBucketReconciler{
		Reconciler: newBaseReconciler(manager, ociobjectstorage.NewObjectStorageBucketServiceManager(provider, credentialClient, scheme, serviceManagerLogger("ObjectStorageBucket")), "ObjectStorageBucket", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupFunctionsApplicationController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.FunctionsApplicationReconciler{
		Reconciler: newBaseReconciler(manager, ocifunctions.NewFunctionsApplicationServiceManager(provider, credentialClient, scheme, serviceManagerLogger("FunctionsApplication")), "FunctionsApplication", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupFunctionsFunctionController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.FunctionsFunctionReconciler{
		Reconciler: newBaseReconciler(manager, ocifunctions.NewFunctionsFunctionServiceManager(provider, credentialClient, scheme, serviceManagerLogger("FunctionsFunction")), "FunctionsFunction", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupDataFlowController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.DataFlowApplicationReconciler{
		Reconciler: newBaseReconciler(manager, ocidataflow.NewDataFlowApplicationServiceManager(provider, credentialClient, scheme, serviceManagerLogger("DataFlowApplication")), "DataFlowApplication", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupContainerInstanceController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.ContainerInstanceReconciler{
		Reconciler: newBaseReconciler(manager, ocicontainerinstance.NewContainerInstanceServiceManager(provider, credentialClient, scheme, serviceManagerLogger("ContainerInstance")), "ContainerInstance", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupComputeInstanceController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.ComputeInstanceReconciler{
		Reconciler: newBaseReconciler(manager, ocicompute.NewComputeInstanceServiceManager(provider, credentialClient, scheme, serviceManagerLogger("ComputeInstance")), "ComputeInstance", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupVCNController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.OciVcnReconciler{
		Reconciler: newBaseReconciler(manager, ocinetworking.NewOciVcnServiceManager(provider, credentialClient, scheme, serviceManagerLogger("OciVcn")), "OciVcn", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupSubnetController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.OciSubnetReconciler{
		Reconciler: newBaseReconciler(manager, ocinetworking.NewOciSubnetServiceManager(provider, credentialClient, scheme, serviceManagerLogger("OciSubnet")), "OciSubnet", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupInternetGatewayController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.OciInternetGatewayReconciler{
		Reconciler: newBaseReconciler(manager, ocinetworking.NewOciInternetGatewayServiceManager(provider, credentialClient, scheme, serviceManagerLogger("OciInternetGateway")), "OciInternetGateway", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupNatGatewayController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.OciNatGatewayReconciler{
		Reconciler: newBaseReconciler(manager, ocinetworking.NewOciNatGatewayServiceManager(provider, credentialClient, scheme, serviceManagerLogger("OciNatGateway")), "OciNatGateway", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupServiceGatewayController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.OciServiceGatewayReconciler{
		Reconciler: newBaseReconciler(manager, ocinetworking.NewOciServiceGatewayServiceManager(provider, credentialClient, scheme, serviceManagerLogger("OciServiceGateway")), "OciServiceGateway", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupDRGController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.OciDrgReconciler{
		Reconciler: newBaseReconciler(manager, ocinetworking.NewOciDrgServiceManager(provider, credentialClient, scheme, serviceManagerLogger("OciDrg")), "OciDrg", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupSecurityListController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.OciSecurityListReconciler{
		Reconciler: newBaseReconciler(manager, ocinetworking.NewOciSecurityListServiceManager(provider, credentialClient, scheme, serviceManagerLogger("OciSecurityList")), "OciSecurityList", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupNetworkSecurityGroupController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.OciNetworkSecurityGroupReconciler{
		Reconciler: newBaseReconciler(manager, ocinetworking.NewOciNetworkSecurityGroupServiceManager(provider, credentialClient, scheme, serviceManagerLogger("OciNetworkSecurityGroup")), "OciNetworkSecurityGroup", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}

func setupRouteTableController(manager ctrl.Manager, provider common.ConfigurationProvider, credentialClient credhelper.CredentialClient, metricsClient *metrics.Metrics) error {
	reconciler := &controllers.OciRouteTableReconciler{
		Reconciler: newBaseReconciler(manager, ocinetworking.NewOciRouteTableServiceManager(provider, credentialClient, scheme, serviceManagerLogger("OciRouteTable")), "OciRouteTable", metricsClient),
	}
	return reconciler.SetupWithManager(manager)
}
