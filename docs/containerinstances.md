# OCI Container Instances

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI Container Instances](https://docs.oracle.com/iaas/Content/container-instances/home.htm), Oracle Cloud Infrastructure's managed container runtime service that runs containers without requiring a Kubernetes cluster.

Using this operator you can create, bind, update, and delete OCI Container Instances directly from your Kubernetes cluster using a `ContainerInstance` custom resource.

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage container instances in your compartment
- A VCN subnet accessible from your cluster nodes

## ContainerInstance CRD

The `ContainerInstance` CRD maps to an OCI Container Instance.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the instance is created |
| `availabilityDomain` | string | Yes | Availability domain where the instance runs |
| `shape` | string | Yes | OCI shape (e.g. `CI.Standard.E4.Flex`) |
| `shapeConfig.ocpus` | float | Yes | Number of OCPUs |
| `shapeConfig.memoryInGBs` | float | Yes | Total memory in GBs |
| `containers` | array | Yes | List of containers (at least one required) |
| `vnics` | array | Yes | List of VNIC configurations (at least one required) |
| `id` | string (OCID) | No | Bind to an existing instance instead of creating one |
| `displayName` | string | No | User-friendly display name. **Required for idempotency** — OSOK uses this to look up existing instances by name, preventing a new instance from being created on every reconcile cycle. |
| `gcPolicy.maxInstances` | integer | No | Maximum number of historical instances to retain (default: 3). Older instances (by creation time) are deleted when the limit is exceeded. Set to `1` for most quota-efficient operation (only the active instance is kept). |
| `faultDomain` | string | No | Fault domain for the instance |
| `gracefulShutdownTimeoutInSeconds` | integer | No | Graceful shutdown timeout |
| `containerRestartPolicy` | string | No | Restart policy: `ALWAYS`, `NEVER`, or `ON_FAILURE` |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Container Fields

Each entry in the `containers` array supports:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `imageUrl` | string | Yes | Container image URL (e.g. `busybox:latest`) |
| `displayName` | string | No | Name for the container |
| `command` | []string | No | Override the container entrypoint |
| `arguments` | []string | No | Arguments for the entrypoint |
| `workingDirectory` | string | No | Working directory inside the container |
| `environmentVariables` | map | No | Additional environment variables |
| `resourceConfig.vcpusLimit` | float | No | Max vCPUs for this container |
| `resourceConfig.memoryLimitInGBs` | float | No | Max memory in GB for this container |

### VNIC Fields

Each entry in the `vnics` array supports:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `subnetId` | string (OCID) | Yes | Subnet for the VNIC |
| `displayName` | string | No | Name for the VNIC |
| `nsgIds` | []string (OCID) | No | NSG OCIDs to associate with the VNIC |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned container instance |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

## Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: ContainerInstance
metadata:
  name: my-container-instance
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  availabilityDomain: "AD-1"
  shape: "CI.Standard.E4.Flex"
  shapeConfig:
    ocpus: 1
    memoryInGBs: 4
  containers:
    - imageUrl: "busybox:latest"
      displayName: "my-container"
      command:
        - "/bin/sh"
      arguments:
        - "-c"
        - "echo hello && sleep 3600"
      environmentVariables:
        MY_ENV: "my-value"
  vnics:
    - subnetId: ocid1.subnet.oc1.iad.xxx
      displayName: "primary-vnic"
  displayName: "my-container-instance"
  containerRestartPolicy: "NEVER"
```

Apply the resource:

```bash
kubectl apply -f my-container-instance.yaml
```

Check status:

```bash
kubectl get containerinstance my-container-instance
kubectl describe containerinstance my-container-instance
```

## Deletion

When you delete a `ContainerInstance` resource, the operator will call the OCI API to delete the underlying container instance.

```bash
kubectl delete containerinstance my-container-instance
```

## Idempotency and displayName

Always set `displayName` on your `ContainerInstance` spec. OSOK's `GetContainerInstanceOcid`
looks up existing instances by display name before creating a new one. Without this field,
OSOK cannot find an existing instance and will create a new OCI Container Instance on every
reconcile cycle, leaking quota.

```yaml
spec:
  displayName: my-container-instance   # Required for idempotency
  compartmentId: ocid1.compartment.oc1..xxx
  # ...
```

## Garbage Collection

When a container instance enters the `FAILED` state, OSOK creates a replacement but cannot
automatically delete the failed instance (it is excluded from the active-instance lookup).
The `gcPolicy` field controls how many historical instances are retained:

```yaml
spec:
  displayName: my-container-instance
  gcPolicy:
    maxInstances: 3   # keep the 3 most recent instances; default 3
```

- **`maxInstances: 1`** — most quota-efficient: only the current active instance is kept.
  Any instance beyond the first (oldest first) is deleted after a new one becomes active.
- **`maxInstances: 3`** (default) — keeps up to 3 instances for debugging failed runs.

## Binding to an Existing Instance

To manage an existing OCI Container Instance through OSOK without creating a new one, set the `id` field:

```yaml
spec:
  id: ocid1.containerinstance.oc1.<region>.xxx
  compartmentId: ocid1.compartment.oc1..xxx
  availabilityDomain: "AD-1"
  shape: "CI.Standard.E4.Flex"
  shapeConfig:
    ocpus: 1
    memoryInGBs: 4
  containers:
    - imageUrl: "busybox:latest"
  vnics:
    - subnetId: ocid1.subnet.oc1.iad.xxx
```
