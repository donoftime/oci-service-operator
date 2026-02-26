# OCI Compute Instance

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI Compute Instances](https://docs.oracle.com/iaas/Content/Compute/home.htm), Oracle Cloud Infrastructure's virtual machine service for running workloads in the cloud.

Using this operator you can create, bind, update, and delete OCI Compute Instances directly from your Kubernetes cluster using a `ComputeInstance` custom resource.

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage compute instances in your compartment
- An image OCID for the desired OS image (e.g. Oracle Linux, Ubuntu)
- A VCN and subnet accessible from your cluster nodes
- A compartment OCID where the instance will be created

## ComputeInstance CRD

The `ComputeInstance` CRD maps to an OCI Compute Instance.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the instance is created |
| `availabilityDomain` | string | Yes | Availability domain where the instance runs |
| `shape` | string | Yes | OCI shape (e.g. `VM.Standard.E4.Flex`) |
| `imageId` | string (OCID) | Yes | OCID of the boot image |
| `subnetId` | string (OCID) | Yes | Subnet for the instance's primary VNIC |
| `shapeConfig.ocpus` | float | No | Number of OCPUs (required for flex shapes) |
| `shapeConfig.memoryInGBs` | float | No | Total memory in GBs (required for flex shapes) |
| `displayName` | string | No | User-friendly display name |
| `id` | string (OCID) | No | Bind to an existing instance instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned compute instance |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

## Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: ComputeInstance
metadata:
  name: my-compute-instance
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  availabilityDomain: "AD-1"
  shape: "VM.Standard.E4.Flex"
  shapeConfig:
    ocpus: 1
    memoryInGBs: 8
  imageId: ocid1.image.oc1.iad.aaaaaaaaxxx
  subnetId: ocid1.subnet.oc1.iad.aaaaaaaaxxx
  displayName: "my-compute-instance"
```

Apply the resource:

```bash
kubectl apply -f my-compute-instance.yaml
```

Check status:

```bash
kubectl get computeinstance my-compute-instance
kubectl describe computeinstance my-compute-instance
```

## Deletion

When you delete a `ComputeInstance` resource, the operator will call the OCI API to terminate the underlying compute instance.

```bash
kubectl delete computeinstance my-compute-instance
```

## Binding to an Existing Instance

To manage an existing OCI Compute Instance through OSOK without creating a new one, set the `id` field:

```yaml
spec:
  id: ocid1.instance.oc1.<region>.xxx
  compartmentId: ocid1.compartment.oc1..xxx
  availabilityDomain: "AD-1"
  shape: "VM.Standard.E4.Flex"
  imageId: ocid1.image.oc1.iad.xxx
  subnetId: ocid1.subnet.oc1.iad.xxx
```
