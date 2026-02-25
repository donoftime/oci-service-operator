# OCI Queue

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI Queue](https://docs.oracle.com/iaas/Content/queue/overview.htm), Oracle Cloud Infrastructure's managed messaging queue service.

Using this operator you can create, bind, update, and delete OCI Queues directly from your Kubernetes cluster using an `OciQueue` custom resource.

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage Queues in your compartment

## OciQueue CRD

The `OciQueue` CRD maps to an OCI Queue resource.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the queue is created |
| `displayName` | string | Yes | User-friendly display name |
| `retentionInSeconds` | integer | No | Message retention period in seconds (min 10) |
| `visibilityInSeconds` | integer | No | Default visibility timeout in seconds (min 1) |
| `timeoutInSeconds` | integer | No | Default polling timeout in seconds (min 1) |
| `deadLetterQueueDeliveryCount` | integer | No | Max delivery attempts before moving to DLQ (0 disables DLQ) |
| `customEncryptionKeyId` | string (OCID) | No | Custom encryption key for message content |
| `id` | string (OCID) | No | Bind to an existing queue instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned Queue |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

### Connection Secret

When a queue is successfully provisioned, OSOK automatically creates a Kubernetes Secret with the same name as the `OciQueue` resource in the same namespace. The secret contains:

| Key | Description |
|-----|-------------|
| `id` | OCID of the queue |
| `messagesEndpoint` | Endpoint URL for producing and consuming messages |
| `displayName` | Display name of the queue |

## Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciQueue
metadata:
  name: my-queue
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: MyQueue
  retentionInSeconds: 86400
  visibilityInSeconds: 30
  timeoutInSeconds: 30
  deadLetterQueueDeliveryCount: 5
```

Apply the resource:

```bash
kubectl apply -f my-queue.yaml
```

Check status:

```bash
kubectl get ociqueue my-queue
kubectl describe ociqueue my-queue
```

Retrieve the connection details:

```bash
kubectl get secret my-queue -o yaml
```

## Deletion

When you delete an `OciQueue` resource, the operator will call the OCI API to delete the underlying queue and remove the associated connection secret.

```bash
kubectl delete ociqueue my-queue
```

## Binding to an Existing Queue

To manage an existing OCI Queue through OSOK without creating a new one, set the `id` field:

```yaml
spec:
  id: ocid1.queue.oc1.<region>.xxx
  compartmentId: ocid1.compartment.oc1..xxx
  displayName: ExistingQueue
```
