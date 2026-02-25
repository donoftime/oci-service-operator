# OCI Container Instances

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI Container Instances](https://docs.oracle.com/iaas/Content/container-instances/home.htm), Oracle Cloud Infrastructure's managed container runtime that lets you run containers without Kubernetes.

Using this operator you can create, bind, and delete container instances directly from your Kubernetes cluster using a `ContainerInstance` custom resource.

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
| `availabilityDomain` | string | Yes | Availability domain (e.g. `Uocm:PHX-AD-1`) |
| `shape` | string | Yes | Container instance shape (e.g. `CI.Standard.E4.Flex`) |
| `ocpus` | float | Yes | Number of OCPUs (minimum 1) |
| `memoryInGBs` | float | No | Memory in gigabytes (defaults to shape minimum) |
| `containers` | array | Yes | List of containers to run (minimum 1) |
| `vnics` | array | Yes | List of VNICs for network access (minimum 1) |
| `displayName` | string | No | User-friendly display name |
| `faultDomain` | string | No | Fault domain placement (e.g. `FAULT-DOMAIN-1`) |
| `containerRestartPolicy` | string | No | Restart policy: `ALWAYS`, `NEVER`, or `ON_FAILURE` |
| `gracefulShutdownTimeoutInSeconds` | integer | No | Graceful shutdown timeout |
| `id` | string (OCID) | No | Bind to an existing container instance instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Container Spec Fields

Each entry in the `containers` array supports:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `imageUrl` | string | Yes | Container image URL (e.g. `docker.io/library/nginx:latest`) |
| `displayName` | string | No | User-friendly name for the container |
| `command` | string array | No | Override the image ENTRYPOINT |
| `arguments` | string array | No | Arguments for the ENTRYPOINT process |
| `workingDirectory` | string | No | Working directory inside the container |
| `environmentVariables` | map | No | Additional environment variables |

### VNIC Spec Fields

Each entry in the `vnics` array supports:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `subnetId` | string (OCID) | Yes | Subnet OCID for this VNIC |
| `displayName` | string | No | User-friendly name for the VNIC |
| `isPublicIpAssigned` | boolean | No | Whether to assign a public IP |
| `nsgIds` | string array | No | Network Security Group OCIDs |

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
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaXXXXXXXXXXXXXXXX
  availabilityDomain: Uocm:PHX-AD-1
  shape: CI.Standard.E4.Flex
  ocpus: 1
  memoryInGBs: 6
  displayName: MyContainerInstance
  containers:
  - imageUrl: docker.io/library/nginx:latest
    displayName: nginx
    environmentVariables:
      NGINX_PORT: "80"
  vnics:
  - subnetId: ocid1.subnet.oc1.iad.XXXXXXXXXXXXXXXXXXXXXXXXXXX
```

## IAM Policies

To allow OSOK to manage container instances, add the following IAM policy to your tenancy:

```
Allow dynamic-group <osok-dynamic-group> to manage container-instances in compartment <compartment-name>
Allow dynamic-group <osok-dynamic-group> to use vnics in compartment <compartment-name>
Allow dynamic-group <osok-dynamic-group> to use subnets in compartment <compartment-name>
```

## Lifecycle

| OCI State | OSOK Condition |
|-----------|----------------|
| CREATING | Provisioning |
| ACTIVE | Active |
| INACTIVE | Active |
| FAILED | Failed |
| DELETING | Terminating |
| DELETED | Terminating |

When a `ContainerInstance` resource is deleted from Kubernetes, OSOK will call the OCI API to delete the underlying container instance.
