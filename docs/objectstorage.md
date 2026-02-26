# OCI Object Storage

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI Object Storage](https://www.oracle.com/cloud/storage/object-storage/), Oracle Cloud Infrastructure's managed object storage service.

Using this operator you can create, bind, update, and delete OCI Object Storage buckets directly from your Kubernetes cluster using an `ObjectStorageBucket` custom resource.

> **Important:** OCI Object Storage buckets do NOT use OCIDs. They are identified by a `namespace/bucketName` composite identifier. The `status.status.ocid` field stores this composite ID.

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage Object Storage in your compartment

## ObjectStorageBucket CRD

The `ObjectStorageBucket` CRD maps to an OCI Object Storage bucket.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the bucket is created |
| `name` | string | Yes | Name of the bucket |
| `namespace` | string | No | OCI Object Storage namespace (auto-resolved from tenancy if empty) |
| `accessType` | string | No | Public access type: `NoPublicAccess`, `ObjectRead`, `ObjectReadWithoutList`, `ObjectWrite` |
| `storageType` | string | No | Storage tier: `Standard` or `Archive` (default: `Standard`) |
| `versioning` | string | No | Object versioning: `Enabled` or `Suspended` |
| `id` | string | No | Bind to an existing bucket using `namespace/bucketName` format |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | Composite identifier `namespace/bucketName` of the provisioned bucket |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

### Connection Secret

When a bucket is successfully provisioned, OSOK automatically creates a Kubernetes Secret with the same name as the `ObjectStorageBucket` resource in the same namespace. The secret contains:

| Key | Description |
|-----|-------------|
| `namespace` | OCI Object Storage namespace |
| `bucketName` | Name of the bucket |
| `apiEndpoint` | Object Storage REST API endpoint for the bucket |

## Example

### Create a new bucket

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: ObjectStorageBucket
metadata:
  name: my-bucket
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  name: my-osok-bucket
  accessType: NoPublicAccess
  storageType: Standard
  versioning: Enabled
```

Apply the resource:

```bash
kubectl apply -f my-bucket.yaml
```

Check status:

```bash
kubectl get objectstoragebucket my-bucket
kubectl describe objectstoragebucket my-bucket
```

Retrieve the connection details:

```bash
kubectl get secret my-bucket -o yaml
```

## Binding to an Existing Bucket

To manage an existing OCI Object Storage bucket through OSOK without creating a new one, set the `id` field with the `namespace/bucketName` composite identifier:

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: ObjectStorageBucket
metadata:
  name: my-bucket
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  name: my-existing-bucket
  id: "mynamespace/my-existing-bucket"
```

## Deletion

When you delete an `ObjectStorageBucket` resource, the operator will call the OCI API to delete the underlying bucket and remove the associated connection secret.

```bash
kubectl delete objectstoragebucket my-bucket
```
