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

# Vault and Key Management
Allow any-user to manage vaults in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}
Allow any-user to manage keys in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# Container Instances
Allow any-user to manage compute-container-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# PostgreSQL DB System
Allow any-user to manage postgres-db-systems in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# Object Storage
Allow any-user to manage object-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# Networking (VCN, Subnets, Gateways, Route Tables, Security Lists)
# Must be 'manage' (not 'use') — OSOK creates and deletes these resources.
Allow any-user to manage virtual-network-family in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}

# Compute Instances
Allow any-user to manage instances in compartment <resources_compartment_name>
  where all {request.principal.type = 'instance',
              request.principal.compartment.id = '<oke_nodes_compartment_ocid>'}
```

---

## Prerequisites

- OKE cluster with OSOK installed ([installation guide](installation.md))
- kro installed in the cluster (`helm install kro oci://ghcr.io/kro-run/kro/kro --namespace kro-system --create-namespace`)
- `kubectl` configured against the cluster
- Your tenancy's Object Storage namespace (`oci os ns get`)
- A **public subnet OCID** (`lbSubnetId`) for the API Gateway — must allow public IPs
- A **compute image OCID** (`imageId`) — Oracle Linux or compatible image in your region
- The VCN and private subnet are created by the platform itself from CIDR blocks you provide

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
new CRD (`OSOKPlatform`) and resolves dependencies automatically.

The platform builds a full VCN topology from scratch — VCN → Internet Gateway →
Route Table → Subnet — then attaches all network-dependent services to that subnet.
kro ensures each layer is active before creating resources that depend on it.

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
      compartmentId: string
      availabilityDomain: string
      # VCN topology — created by the platform
      cidrBlock: string           # VCN CIDR, e.g. "10.0.0.0/16"
      subnetCidrBlock: string     # Private subnet CIDR, e.g. "10.0.1.0/24"
      # Pre-existing public subnet for the API Gateway (must allow public IPs)
      lbSubnetId: string
      objectStorageNamespace: string
      mysqlCredentialsSecret: string
      adbAdminPasswordSecret: string
      adbDbName: string           # No hyphens — used as DB name (e.g. myplatformadb)
      nosqlTableName: string      # No hyphens — used in DDL (e.g. myplatform_events)
      containerImageUrl: string
      imageId: string             # Boot image OCID for the Compute VM

  resources:

    # ── 1. Vault ─────────────────────────────────────────────────────────────
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
          dbName: ${schema.spec.adbDbName}
          dbWorkload: OLTP
          cpuCoreCount: 1
          dataStorageSizeInTBs: 1
          isAutoScalingEnabled: false
          isFreeTier: false
          adminPassword:
            secret:
              secretName: ${schema.spec.adbAdminPasswordSecret}

    # ── 6. VCN ────────────────────────────────────────────────────────────────
    # Foundation of the network topology. Internet Gateway and Route Table
    # depend on this OCID.
    - id: vcn
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: OciVcn
        metadata:
          name: ${schema.metadata.name}-vcn
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-vcn
          cidrBlock: ${schema.spec.cidrBlock}
          dnsLabel: platform

    # ── 7. Internet Gateway ───────────────────────────────────────────────────
    # Provides outbound internet access. Route Table depends on this OCID.
    - id: internetGateway
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: OciInternetGateway
        metadata:
          name: ${schema.metadata.name}-igw
        spec:
          compartmentId: ${schema.spec.compartmentId}
          vcnId: ${vcn.status.status.ocid}
          displayName: ${schema.metadata.name}-igw
          isEnabled: true
      readyWhen:
        - ${internetGateway.status.status.conditions[-1].type == 'Active'}

    # ── 8. Route Table ────────────────────────────────────────────────────────
    # Default route via Internet Gateway. Subnet depends on this OCID.
    - id: routeTable
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: OciRouteTable
        metadata:
          name: ${schema.metadata.name}-rt
        spec:
          compartmentId: ${schema.spec.compartmentId}
          vcnId: ${vcn.status.status.ocid}
          displayName: ${schema.metadata.name}-rt
          routeRules:
            - destination: "0.0.0.0/0"
              networkEntityId: ${internetGateway.status.status.ocid}
              destinationType: CIDR_BLOCK
      readyWhen:
        - ${routeTable.status.status.conditions[-1].type == 'Active'}

    # ── 9. Subnet ─────────────────────────────────────────────────────────────
    # Private subnet (no public IPs). All network-attached services use this.
    - id: subnet
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: OciSubnet
        metadata:
          name: ${schema.metadata.name}-subnet
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-subnet
          vcnId: ${vcn.status.status.ocid}
          cidrBlock: ${schema.spec.subnetCidrBlock}
          dnsLabel: private
          prohibitPublicIpOnVnic: true
          routeTableId: ${routeTable.status.status.ocid}

    # ── 10. MySQL DB System ───────────────────────────────────────────────────
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
          subnetId: ${subnet.status.status.ocid}
          availabilityDomain: ${schema.spec.availabilityDomain}
          mysqlVersion: "8.0.42"
          dataStorageSizeInGBs: 50
          adminUsername:
            secret:
              secretName: ${schema.spec.mysqlCredentialsSecret}
          adminPassword:
            secret:
              secretName: ${schema.spec.mysqlCredentialsSecret}

    # ── 11. PostgreSQL DB System ──────────────────────────────────────────────
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
          subnetId: ${subnet.status.status.ocid}
          storageType: HighPerformance
          instanceCount: 1
          instanceOcpuCount: 2
          instanceMemoryInGBs: 32

    # ── 12. NoSQL Table ───────────────────────────────────────────────────────
    - id: nosqlTable
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: NoSQLDatabase
        metadata:
          name: ${schema.metadata.name}-nosql
        spec:
          compartmentId: ${schema.spec.compartmentId}
          name: ${schema.spec.nosqlTableName}
          ddlStatement: >-
            CREATE TABLE IF NOT EXISTS ${schema.spec.nosqlTableName}
            (id STRING, ts TIMESTAMP(3), payload JSON, PRIMARY KEY(id))
          tableLimits:
            maxReadUnits: 50
            maxWriteUnits: 50
            maxStorageInGBs: 1

    # ── 13. OCI Cache with Redis ──────────────────────────────────────────────
    - id: redisCluster
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: RedisCluster
        metadata:
          name: ${schema.metadata.name}-redis
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-redis
          softwareVersion: V7_0_5
          nodeCount: 3
          nodeMemoryInGBs: 2
          subnetId: ${subnet.status.status.ocid}

    # ── 14. OpenSearch Cluster ────────────────────────────────────────────────
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
          masterNodeHostMemoryGB: 32
          dataNodeCount: 3
          dataNodeHostType: FLEX
          dataNodeHostOcpuCount: 4
          dataNodeHostMemoryGB: 64
          dataNodeStorageGB: 100
          opendashboardNodeCount: 1
          opendashboardNodeHostOcpuCount: 2
          opendashboardNodeHostMemoryGB: 32
          vcnId: ${vcn.status.status.ocid}
          vcnCompartmentId: ${schema.spec.compartmentId}
          subnetId: ${subnet.status.status.ocid}
          subnetCompartmentId: ${schema.spec.compartmentId}

    # ── 15. API Gateway ───────────────────────────────────────────────────────
    # Uses a pre-existing public/LB subnet (lbSubnetId) — the platform subnet
    # prohibits public IPs, so API Gateway needs its own public-facing subnet.
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
          subnetId: ${schema.spec.lbSubnetId}

    # ── 16. API Gateway Deployment ────────────────────────────────────────────
    # Depends on apiGateway. Serves a stock health-check response.
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

    # ── 17. Container Instance ────────────────────────────────────────────────
    # displayName is required for idempotency: OSOK uses it to look up existing
    # instances so that reconcile does not create a new instance on every cycle.
    # gcPolicy (optional): controls how many historical instances are retained.
    #   gcPolicy:
    #     maxInstances: 3  # keep the 3 most recent instances; default 3
    # Setting maxInstances: 1 is most quota-efficient (only the active instance).
    - id: containerInstance
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: ContainerInstance
        metadata:
          name: ${schema.metadata.name}-ci
        spec:
          compartmentId: ${schema.spec.compartmentId}
          availabilityDomain: ${schema.spec.availabilityDomain}
          displayName: ${schema.metadata.name}-ci
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
            - subnetId: ${subnet.status.status.ocid}
              displayName: primary-vnic
          containerRestartPolicy: ON_FAILURE

    # ── 18. Compute Instance ──────────────────────────────────────────────────
    # A full VM in the platform subnet. imageId must be an OCI image OCID
    # valid in your region (e.g. Oracle Linux 8).
    - id: computeInstance
      template:
        apiVersion: oci.oracle.com/v1beta1
        kind: ComputeInstance
        metadata:
          name: ${schema.metadata.name}-vm
        spec:
          compartmentId: ${schema.spec.compartmentId}
          displayName: ${schema.metadata.name}-vm
          availabilityDomain: ${schema.spec.availabilityDomain}
          shape: VM.Standard.E4.Flex
          shapeConfig:
            ocpus: 1
            memoryInGBs: 16
          imageId: ${schema.spec.imageId}
          subnetId: ${subnet.status.status.ocid}
```

---

## Deploying the Platform

Once the `ResourceGraphDefinition` is applied, kro registers `OSOKPlatform` as a
new CRD. Create an instance to provision the full platform:

```bash
# Apply the RGD
kubectl apply -f kro-platform-rgd.yaml

# Verify kro registered it
kubectl get resourcegraphdefinitions osok-platform
```

Create the instance. Substitute your real values — note `adbDbName` and `nosqlTableName`
must not contain hyphens (OCI restriction):

```yaml
# kro-platform-instance.yaml
apiVersion: kro.run/v1alpha1
kind: OSOKPlatform
metadata:
  name: my-platform
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  availabilityDomain: IqDk:US-ASHBURN-AD-1

  # VCN topology — created from scratch
  cidrBlock: "10.0.0.0/16"
  subnetCidrBlock: "10.0.1.0/24"

  # Pre-existing public subnet for the API Gateway
  lbSubnetId: ocid1.subnet.oc1.iad.xxx

  objectStorageNamespace: mytenancy
  mysqlCredentialsSecret: osok-platform-mysql-creds
  adbAdminPasswordSecret: osok-platform-adb-creds

  adbDbName: myplatformadb         # no hyphens
  nosqlTableName: myplatform_events  # no hyphens

  containerImageUrl: iad.ocir.io/mytenancy/my-repo/myapp:latest

  # Oracle Linux 8 image OCID — find yours with:
  # oci compute image list --compartment-id <ocid> --operating-system "Oracle Linux"
  imageId: ocid1.image.oc1.iad.xxx
```

```bash
kubectl apply -f kro-platform-instance.yaml
```

---

## Verifying Resources

Watch the overall platform status:

```bash
kubectl get osokplatform my-platform -w
```

Check individual resources as they come up:

```bash
kubectl get \
  ocivault,objectstoragebucket,stream,ociqueue,autonomousdatabases,\
  ocivcn,ociinternetgateway,ociroutetable,ocisubnet,\
  mysqldbsystem,postgresdbsystem,nosqldatabase,rediscluster,opensearchcluster,\
  apigateway,apigatewaydeployment,containerinstance,computeinstance \
  -o custom-columns='KIND:.kind,NAME:.metadata.name,STATUS:.status.status.conditions[-1].type'
```

For services that create Kubernetes Secrets (MySQL, Redis, Queue, PostgreSQL,
Object Storage), retrieve connection details:

```bash
kubectl get secret my-platform-mysql -o yaml
kubectl get secret my-platform-redis -o yaml
kubectl get secret my-platform-queue -o yaml
kubectl get secret my-platform-postgres -o yaml
kubectl get secret my-platform-bucket -o yaml
```

The API Gateway endpoint is in the gateway status message:

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
queue               ─┤  (independent — run in parallel)
adb                 ─┤
apiGateway ──────────┤──► apiGatewayDeployment
                     │
vcn ─────────────────┤──► internetGateway ──► routeTable ──► subnet ──► mysql
                     │                                              ├──► postgres
                     │                                              ├──► redisCluster
                     │                                              ├──► opensearch
                     │                                              ├──► containerInstance
                     │                                              └──► computeInstance
                     │
nosqlTable ──────────┘
```

The VCN topology chain (`vcn → internetGateway → routeTable → subnet`) must
complete before any subnet-attached service is created. kro handles this
automatically by tracking the `${resource.status.status.ocid}` references.

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
- **VCN resources**: Subnet must be deleted before Route Table, Internet Gateway,
  and VCN — kro/OSOK handle this order automatically via finalizers

Remove the RGD when no instances remain:

```bash
kubectl delete resourcegraphdefinition osok-platform
```
