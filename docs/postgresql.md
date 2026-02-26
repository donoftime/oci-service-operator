# OCI Database with PostgreSQL

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI Database with PostgreSQL](https://www.oracle.com/cloud/database/), Oracle Cloud Infrastructure's managed PostgreSQL database service.

Using this operator you can create, bind, update, and delete OCI PostgreSQL DB systems directly from your Kubernetes cluster using a `PostgresDbSystem` custom resource.

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage PostgreSQL DB systems in your compartment
- A VCN subnet accessible from your cluster nodes

## PostgresDbSystem CRD

The `PostgresDbSystem` CRD maps to an OCI Database with PostgreSQL DB system.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the DB system is created |
| `displayName` | string | Yes | User-friendly display name |
| `dbVersion` | string | Yes | PostgreSQL version (e.g. `14.10`) |
| `shape` | string | Yes | Instance shape for DB nodes (e.g. `VM.Standard.E4.Flex`) |
| `subnetId` | string (OCID) | Yes | Subnet for the DB system endpoints |
| `storageType` | string | Yes | Storage tier: `HighPerformance` or `Balanced` |
| `description` | string | No | Optional description of the DB system |
| `instanceCount` | integer | No | Number of DB instance nodes (defaults to 1) |
| `instanceOcpuCount` | integer | No | OCPUs available to each instance node |
| `instanceMemoryInGBs` | integer | No | Memory available to each instance node, in gigabytes |
| `id` | string (OCID) | No | Bind to an existing DB system instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned PostgreSQL DB system |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

### Connection Secret

When a DB system is successfully provisioned, OSOK automatically creates a Kubernetes Secret with the same name as the `PostgresDbSystem` resource in the same namespace. The secret contains:

| Key | Description |
|-----|-------------|
| `id` | OCID of the provisioned DB system |
| `displayName` | Display name of the DB system |
| `primaryEndpoint` | Private IP of the primary DB endpoint (if available) |
| `port` | PostgreSQL port (5432) |

## Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: PostgresDbSystem
metadata:
  name: my-postgres-db
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: MyPostgresDB
  dbVersion: "14.10"
  shape: "VM.Standard.E4.Flex"
  subnetId: ocid1.subnet.oc1.iad.xxx
  storageType: HighPerformance
  instanceCount: 1
  instanceOcpuCount: 2
  instanceMemoryInGBs: 32
```

Apply the resource:

```bash
kubectl apply -f my-postgres-db.yaml
```

Check status:

```bash
kubectl get postgresdbsystem my-postgres-db
kubectl describe postgresdbsystem my-postgres-db
```

Read the connection secret:

```bash
kubectl get secret my-postgres-db -o jsonpath='{.data.id}' | base64 -d
kubectl get secret my-postgres-db -o jsonpath='{.data.primaryEndpoint}' | base64 -d
kubectl get secret my-postgres-db -o jsonpath='{.data.port}' | base64 -d
```

## Deletion

To delete the DB system, delete the Kubernetes resource:

```bash
kubectl delete postgresdbsystem my-postgres-db
```

OSOK will call the OCI API to delete the DB system and clean up the connection secret.

## Binding to an Existing DB System

To bind OSOK management to a pre-existing OCI PostgreSQL DB system, set the `id` field in the spec:

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: PostgresDbSystem
metadata:
  name: existing-postgres-db
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: ExistingPostgresDB
  dbVersion: "14.10"
  shape: "VM.Standard.E4.Flex"
  subnetId: ocid1.subnet.oc1.iad.xxx
  storageType: HighPerformance
  id: ocid1.postgresql.oc1.iad.existing-ocid
```

When `id` is set, OSOK will bind to the existing DB system and manage updates and deletion through it.
