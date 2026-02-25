# OCI NoSQL Database

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI NoSQL Database](https://docs.oracle.com/iaas/Content/nosql/home.htm), Oracle Cloud Infrastructure's managed NoSQL table service.

Using this operator you can create, bind, update, and delete NoSQL tables directly from your Kubernetes cluster using a `NoSQLDatabase` custom resource.

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage NoSQL tables in your compartment

## NoSQLDatabase CRD

The `NoSQLDatabase` CRD maps to an OCI NoSQL table.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the table is created |
| `name` | string | Yes | Table name (immutable after creation) |
| `ddlStatement` | string | Yes | Complete `CREATE TABLE` DDL statement |
| `tableLimits.maxReadUnits` | integer | Yes* | Maximum read throughput (provisioned mode) |
| `tableLimits.maxWriteUnits` | integer | Yes* | Maximum write throughput (provisioned mode) |
| `tableLimits.maxStorageInGBs` | integer | Yes* | Maximum storage in gigabytes |
| `id` | string (OCID) | No | Bind to an existing table instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

*Required for provisioned capacity mode.

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned NoSQL table |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

## Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: NoSQLDatabase
metadata:
  name: my-nosql-table
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  name: MyTable
  ddlStatement: "CREATE TABLE IF NOT EXISTS MyTable (id INTEGER, name STRING, PRIMARY KEY(id))"
  tableLimits:
    maxReadUnits: 50
    maxWriteUnits: 50
    maxStorageInGBs: 25
```

Apply the resource:

```bash
kubectl apply -f my-nosql-table.yaml
```

Check status:

```bash
kubectl get nosqldatabase my-nosql-table
kubectl describe nosqldatabase my-nosql-table
```

## Deletion

When you delete a `NoSQLDatabase` resource, the operator will call the OCI API to delete the underlying NoSQL table.

```bash
kubectl delete nosqldatabase my-nosql-table
```

## Binding to an Existing Table

To manage an existing OCI NoSQL table through OSOK without creating a new one, set the `id` field:

```yaml
spec:
  id: ocid1.nosqltable.oc1.<region>.xxx
  compartmentId: ocid1.compartment.oc1..xxx
  name: ExistingTable
  ddlStatement: "CREATE TABLE IF NOT EXISTS ExistingTable (id INTEGER, name STRING, PRIMARY KEY(id))"
```

## Updating a Table

To update table limits or DDL, modify the spec and apply the resource again. The operator will call the OCI UpdateTable API:

```yaml
spec:
  tableLimits:
    maxReadUnits: 100
    maxWriteUnits: 100
    maxStorageInGBs: 50
```
