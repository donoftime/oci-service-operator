# OSOK with kro: Complete Platform Walkthrough

This guide shows how to use [kro (Kube Resource Orchestrator)](https://kro.run) to
deploy all OSOK-supported OCI services from a single Kubernetes manifest. You define
one `ResourceGraphDefinition` that fans out to create every service; kro handles
dependency ordering and lifecycle.

---

## Required IAM Policies

Apply these policies in the OCI Console or CLI before deploying OSOK. They grant OKE
node instances permission to manage each service without requiring dynamic groups.

Replace the two placeholders throughout:
- `<oke_nodes_compartment_ocid>` — OCID of the compartment where your OKE node pool runs
- `<resources_compartment_name>` — name of the compartment where OSOK will create resources

```
# Autonomous Database
Allow any-user to manage autonomous-database-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# Oracle Streaming Service
Allow any-user to manage stream-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# MySQL DB System
Allow any-user to manage mysql-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# OCI Cache with Redis
Allow any-user to manage redis-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# OCI Search with OpenSearch
Allow any-user to manage opensearch-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# OCI Queue
Allow any-user to manage queues in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# API Gateway
Allow any-user to manage api-gateway-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# NoSQL Database
Allow any-user to manage nosql-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# Functions (also needs VCN read for subnet validation)
Allow any-user to manage functions-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}
Allow any-user to use virtual-network-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# Vault and Key Management
Allow any-user to manage vaults in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}
Allow any-user to manage keys in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# Container Instances
Allow any-user to manage container-instances in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# PostgreSQL DB System
Allow any-user to manage postgresql-db-systems in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# Object Storage
Allow any-user to manage object-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# Data Flow (also needs object storage access for app files and logs)
Allow any-user to manage dataflow-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}
```

---

## Prerequisites

- OKE cluster with OSOK installed ([installation guide](installation.md))
- kro installed in the cluster (`helm install kro oci://ghcr.io/kro-run/kro/kro --namespace kro-system --create-namespace`)
- `kubectl` configured against the cluster
- Your tenancy's Object Storage namespace (`oci os ns get`)
- A subnet accessible from your OKE nodes (for MySQL, PostgreSQL, Redis, OpenSearch, API Gateway, Functions, Container Instances)
- A VCN OCID (for OpenSearch)
- An OCIR image pushed for your Functions function
- An OCI Object Storage URI for your Data Flow application file (e.g. `oci://my-bucket@my-namespace/app.py`)

---

## Pre-create Credential Secrets

OSOK reads database credentials from Kubernetes Secrets. Create these before deploying
the platform:

```bash
# MySQL admin credentials
kubectl create secret generic osok-platform-mysql-creds \
  --from-literal=username=admin \
  --from-literal=password='<your-mysql-password>'

# Autonomous Database admin password
kubectl create secret generic osok-platform-adb-creds \
  --from-literal=password='<your-adb-password>'
```

---

## ResourceGraphDefinition

This single manifest defines a reusable platform blueprint. kro registers it as a
new CRD (`OSOKPlatform`) and resolves dependencies automatically — for example,
`FunctionsFunction` waits for `FunctionsApplication` to be provisioned before
kro creates it, because its `applicationId` field references the application's
status OCID.

```yaml
apiVersion: kro.run/v1alpha1
kind: ResourceGraphDefinition
metadata:
  name: osok-platform
spec:
  schema:
    apiVersion: v1alpha1
    kind: OSOKPlatform
    spec:
      properties:
        # ── Core ────────────────────────────────────────────────────────────
        compartmentId:
          type: string
          description: "OCID of the OCI compartment for all resources"
        availabilityDomain:
          type: string
          description: "Availability domain (e.g. AD-1)"
        subnetId:
          type: string
          description: "Subnet OCID for network-attached services"
        vcnId:
          type: string
          description: "VCN OCID (required by OpenSearch)"
        objectStorageNamespace:
          type: string
          description: "Tenancy Object Storage namespace (from: oci os ns get)"

        # ── Credentials ─────────────────────────────────────────────────────
        mysqlCredentialsSecret:
          type: string
          description: "K8s secret name with 'username' and 'password' keys for MySQL"
        adbAdminPasswordSecret:
          type: string
          description: "K8s secret name with 'password' key for Autonomous Database"

        # ── Compute / Serverless ────────────────────────────────────────────
        functionsImageUrl:
          type: string
          description: "OCIR image URL for the Functions function (e.g. iad.ocir.io/mytenancy/myrepo/fn:latest)"
        containerImageUrl:
          type: string
          description: "Container image URL for the Container Instance"

        # ── Analytics ───────────────────────────────────────────────────────
        dataflowFileUri:
          type: string
          description: "OCI Object Storage URI of the Data Flow application file (e.g. oci://bucket@namespace/app.py)"

      required:
        - compartmentId
        - availabilityDomain
        - subnetId
        - vcnId
        - objectStorageNamespace
        - mysqlCredentialsSecret
        - adbAdminPasswordSecret
        - functionsImageUrl
        - containerImageUrl
        - dataflowFileUri

  resources:

    # ── 1. Vault ─────────────────────────────────────────────────────────────
    # Independent. Creates a DEFAULT vault with a managed AES-256 encryption key.
    - id: vault
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: OciVault
        metadata:
          name: ${schema.metadata.name}-vault
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-vault
          vaultType: DEFAULT
          key:
            displayName: ${schema.metadata.name}-key
            keyShape:
              algorithm: AES
              length: 32

    # ── 2. Object Storage Bucket ──────────────────────────────────────────────
    # Independent. Used by Data Flow for logs.
    - id: objectStorageBucket
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: ObjectStorageBucket
        metadata:
          name: ${schema.metadata.name}-bucket
        spec:
          compartmentId: ${schema.spec.compartmentId}
          name: ${schema.metadata.name}-bucket
          accessType: NoPublicAccess
          storageType: Standard
          versioning: Enabled

    # ── 3. Oracle Streaming Service ───────────────────────────────────────────
    - id: stream
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: Stream
        metadata:
          name: ${schema.metadata.name}-stream
        spec:
          compartmentId: ${schema.spec.compartmentId}
          name: ${schema.metadata.name}-stream
          partitions: 1
          retentionInHours: 24

    # ── 4. OCI Queue ──────────────────────────────────────────────────────────
    - id: queue
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: OciQueue
        metadata:
          name: ${schema.metadata.name}-queue
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-queue
          retentionInSeconds: 86400
          visibilityInSeconds: 30
          deadLetterQueueDeliveryCount: 3

    # ── 5. Autonomous Database ────────────────────────────────────────────────
    - id: adb
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: AutonomousDatabases
        metadata:
          name: ${schema.metadata.name}-adb
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-adb
          dbName: ${schema.metadata.name}adb
          dbWorkload: OLTP
          cpuCoreCount: 1
          dataStorageSizeInTBs: 1
          isAutoScalingEnabled: false
          isFreeTier: false
          adminPassword:
            secret:
              secretName: ${schema.spec.adbAdminPasswordSecret}

    # ── 6. MySQL DB System ────────────────────────────────────────────────────
    - id: mysql
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: MySqlDbSystem
        metadata:
          name: ${schema.metadata.name}-mysql
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-mysql
          shapeName: MySQL.VM.Standard.E3.1.8GB
          subnetId: ${schema.spec.subnetId}
          availabilityDomain: ${schema.spec.availabilityDomain}
          mysqlVersion: "8.0"
          dataStorageSizeInGBs: 50
          adminUsername:
            secret:
              secretName: ${schema.spec.mysqlCredentialsSecret}
          adminPassword:
            secret:
              secretName: ${schema.spec.mysqlCredentialsSecret}

    # ── 7. PostgreSQL DB System ───────────────────────────────────────────────
    - id: postgres
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: PostgresDbSystem
        metadata:
          name: ${schema.metadata.name}-postgres
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-postgres
          dbVersion: "14.10"
          shape: VM.Standard.E4.Flex
          subnetId: ${schema.spec.subnetId}
          storageType: HighPerformance
          instanceCount: 1
          instanceOcpuCount: 2
          instanceMemoryInGBs: 32

    # ── 8. NoSQL Table ────────────────────────────────────────────────────────
    - id: nosqlTable
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: NoSQLDatabase
        metadata:
          name: ${schema.metadata.name}-nosql
        spec:
          compartmentId: ${schema.spec.compartmentId}
          name: ${schema.metadata.name}_events
          ddlStatement: >-
            CREATE TABLE IF NOT EXISTS ${schema.metadata.name}_events
            (id STRING, ts TIMESTAMP(3), payload JSON, PRIMARY KEY(id))
          tableLimits:
            maxReadUnits: 50
            maxWriteUnits: 50
            maxStorageInGBs: 1

    # ── 9. OCI Cache with Redis ────────────────────────────────────────────────
    - id: redisCluster
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: RedisCluster
        metadata:
          name: ${schema.metadata.name}-redis
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-redis
          softwareVersion: REDIS_7_0
          nodeCount: 3
          nodeMemoryInGBs: 1
          subnetId: ${schema.spec.subnetId}

    # ── 10. OpenSearch Cluster ────────────────────────────────────────────────
    - id: opensearch
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: OpenSearchCluster
        metadata:
          name: ${schema.metadata.name}-opensearch
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-opensearch
          softwareVersion: "2.11.0"
          masterNodeCount: 3
          masterNodeHostType: FLEX
          masterNodeHostOcpuCount: 2
          masterNodeHostMemoryGB: 16
          dataNodeCount: 3
          dataNodeHostType: FLEX
          dataNodeHostOcpuCount: 4
          dataNodeHostMemoryGB: 32
          dataNodeStorageGB: 100
          opendashboardNodeCount: 1
          opendashboardNodeHostOcpuCount: 2
          opendashboardNodeHostMemoryGB: 16
          vcnId: ${schema.spec.vcnId}
          vcnCompartmentId: ${schema.spec.compartmentId}
          subnetId: ${schema.spec.subnetId}
          subnetCompartmentId: ${schema.spec.compartmentId}

    # ── 11. API Gateway ───────────────────────────────────────────────────────
    # The deployment (resource 12) is created after this because it references
    # this gateway's status OCID.
    - id: apiGateway
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: ApiGateway
        metadata:
          name: ${schema.metadata.name}-apigw
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-apigw
          endpointType: PUBLIC
          subnetId: ${schema.spec.subnetId}

    # ── 12. Functions Application ─────────────────────────────────────────────
    # The function (resource 13) is created after this because it references
    # this application's status OCID.
    - id: functionsApp
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: FunctionsApplication
        metadata:
          name: ${schema.metadata.name}-fn-app
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-fn-app
          subnetIds:
            - ${schema.spec.subnetId}

    # ── 13. Functions Function ────────────────────────────────────────────────
    # Depends on functionsApp: kro waits for functionsApp.status.status.ocid
    # to be populated before creating this resource.
    - id: functionsFunction
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: FunctionsFunction
        metadata:
          name: ${schema.metadata.name}-fn
        spec:
          compartmentId: ${schema.spec.compartmentId}
          applicationId: ${functionsApp.status.status.ocid}
          displayName: ${schema.metadata.name}-fn
          image: ${schema.spec.functionsImageUrl}
          memoryInMBs: 256
          timeoutInSeconds: 60

    # ── 14. API Gateway Deployment ────────────────────────────────────────────
    # Depends on both apiGateway and functionsFunction (two status OCID refs).
    # kro creates this only after both are Active.
    - id: apiGatewayDeployment
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: ApiGatewayDeployment
        metadata:
          name: ${schema.metadata.name}-apigw-deploy
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-apigw-deploy
          gatewayId: ${apiGateway.status.status.ocid}
          pathPrefix: /v1
          routes:
            - path: /health
              methods:
                - GET
              backend:
                type: STOCK_RESPONSE_BACKEND
                status: 200
                body: '{"status":"ok"}'
            - path: /process
              methods:
                - POST
              backend:
                type: ORACLE_FUNCTIONS_BACKEND
                functionId: ${functionsFunction.status.status.ocid}

    # ── 15. Container Instance ────────────────────────────────────────────────
    - id: containerInstance
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: ContainerInstance
        metadata:
          name: ${schema.metadata.name}-ci
        spec:
          compartmentId: ${schema.spec.compartmentId}
          availabilityDomain: ${schema.spec.availabilityDomain}
          shape: CI.Standard.E4.Flex
          shapeConfig:
            ocpus: 1
            memoryInGBs: 4
          containers:
            - imageUrl: ${schema.spec.containerImageUrl}
              displayName: app
              environmentVariables:
                COMPARTMENT_ID: ${schema.spec.compartmentId}
          vnics:
            - subnetId: ${schema.spec.subnetId}
              displayName: primary-vnic
          containerRestartPolicy: ON_FAILURE

    # ── 16. Data Flow Application ─────────────────────────────────────────────
    # Uses the Object Storage bucket (resource 2) for log output.
    # The logsBucketUri is constructed from the bucket name and tenancy namespace.
    - id: dataflowApp
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: DataFlowApplication
        metadata:
          name: ${schema.metadata.name}-dataflow
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-dataflow
          language: PYTHON
          driverShape: VM.Standard2.1
          executorShape: VM.Standard2.1
          numExecutors: 1
          sparkVersion: "3.2.1"
          fileUri: ${schema.spec.dataflowFileUri}
          logsBucketUri: "oci://${schema.metadata.name}-bucket@${schema.spec.objectStorageNamespace}"
```

---

## Deploying the Platform

Once the `ResourceGraphDefinition` is applied, kro registers `OSOKPlatform` as a
new CRD. Create an instance to provision the full platform:

```bash
# Apply the RGD
kubectl apply -f osok-platform-rgd.yaml

# Verify kro registered it
kubectl get resourcegraphdefinitions osok-platform
```

```yaml
# osok-platform-instance.yaml
apiVersion: kro.run/v1alpha1
kind: OSOKPlatform
metadata:
  name: my-platform
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  availabilityDomain: AD-1
  subnetId: ocid1.subnet.oc1.iad.xxx
  vcnId: ocid1.vcn.oc1.iad.xxx
  objectStorageNamespace: mytenancy

  mysqlCredentialsSecret: osok-platform-mysql-creds
  adbAdminPasswordSecret: osok-platform-adb-creds

  functionsImageUrl: iad.ocir.io/mytenancy/my-repo/my-fn:latest
  containerImageUrl: busybox:latest
  dataflowFileUri: "oci://my-dataflow-bucket@mytenancy/app.py"
```

```bash
kubectl apply -f osok-platform-instance.yaml
```

---

## Verifying Resources

Watch the overall platform status:

```bash
kubectl get osokplatform my-platform -w
```

Check individual resources as they come up:

```bash
# All OSOK resources in the namespace
kubectl get ocivault,objectstoragebucket,stream,ociqueue,autonomousdatabases,\
  mysqldbsystem,postgresdbsystem,nosqldatabase,rediscluster,opensearchcluster,\
  apigateway,apigatewaydeployment,functionsapplication,functionsfunction,\
  containerinstance,dataflowapplication -n default

# Detailed status of a specific resource
kubectl describe ocivault my-platform-vault
kubectl describe functionsfunction my-platform-fn
```

For services that create Kubernetes Secrets (Vault, Redis, Queue, PostgreSQL,
Object Storage), retrieve the connection details:

```bash
kubectl get secret my-platform-vault -o yaml
kubectl get secret my-platform-redis -o yaml
kubectl get secret my-platform-queue -o yaml
kubectl get secret my-platform-postgres -o yaml
kubectl get secret my-platform-bucket -o yaml
```

The API Gateway endpoint is available from the gateway status:

```bash
kubectl get apigateway my-platform-apigw \
  -o jsonpath='{.status.status.conditions[?(@.type=="Active")].message}'
```

---

## Dependency Graph

kro resolves resources in parallel where possible and serializes where dependencies
exist. The graph for this platform is:

```
vault               ─┐
objectStorageBucket ─┤
stream              ─┤
queue               ─┤  (all independent, run in parallel)
adb                 ─┤
mysql               ─┤
postgres            ─┤
nosqlTable          ─┤
redisCluster        ─┤
opensearch          ─┤
                     │
apiGateway ──────────┤──► apiGatewayDeployment
                     │         ▲
functionsApp ────────┤──► functionsFunction ─┘
                     │
containerInstance   ─┤
dataflowApp ─────────┘
```

---

## Cleanup

Delete the platform instance to trigger deletion of all OCI resources:

```bash
kubectl delete osokplatform my-platform
```

OSOK finalizers ensure each OCI resource is deleted before the Kubernetes object
is removed. Note that some services impose deletion delays:
- **Vault**: OCI schedules deletion with a minimum 7-day grace period
- **ADB**: Terminates immediately but may take a few minutes
- **OpenSearch / MySQL / PostgreSQL**: Active deletion, may take several minutes

Remove the RGD when no instances remain:

```bash
kubectl delete resourcegraphdefinition osok-platform
```
