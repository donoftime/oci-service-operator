# OCI Networking (VCN and Subnet)

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports OCI Networking resources, allowing you to manage [Virtual Cloud Networks (VCN)](https://docs.oracle.com/iaas/Content/Network/Tasks/managingVCNs.htm) and [Subnets](https://docs.oracle.com/iaas/Content/Network/Tasks/managingsubnets.htm) directly from your Kubernetes cluster.

Using this operator you can create, bind, and delete OCI VCNs and Subnets using the `OciVcn` and `OciSubnet` custom resources. These resources are typically created before provisioning compute or container workloads that require network connectivity.

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage networking resources in your compartment
- A compartment OCID where the VCN and subnet will be created

## OciVcn CRD

The `OciVcn` CRD maps to an OCI Virtual Cloud Network.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the VCN is created |
| `displayName` | string | Yes | User-friendly display name |
| `cidrBlock` | string | Yes | CIDR block for the VCN (e.g. `10.0.0.0/16`) |
| `dnsLabel` | string | No | DNS label for the VCN's internal hostname resolution |
| `id` | string (OCID) | No | Bind to an existing VCN instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned VCN |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

## OciSubnet CRD

The `OciSubnet` CRD maps to an OCI Subnet within a VCN.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the subnet is created |
| `displayName` | string | Yes | User-friendly display name |
| `vcnId` | string (OCID) | Yes | OCID of the VCN that contains this subnet |
| `cidrBlock` | string | Yes | CIDR block for the subnet (must be within the VCN CIDR) |
| `availabilityDomain` | string | No | Availability domain for an AD-specific subnet (omit for regional) |
| `dnsLabel` | string | No | DNS label for hostname resolution within the subnet |
| `prohibitPublicIpOnVnic` | bool | No | When true, VNICs in this subnet cannot have public IPs (private subnet) |
| `routeTableId` | string (OCID) | No | OCID of the route table the subnet uses |
| `securityListIds` | []string (OCID) | No | List of security list OCIDs associated with the subnet |
| `id` | string (OCID) | No | Bind to an existing subnet instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned subnet |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

## Examples

### Create a VCN

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciVcn
metadata:
  name: my-vcn
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: my-vcn
  cidrBlock: "10.0.0.0/16"
  dnsLabel: myvcn
```

Apply and check status:

```bash
kubectl apply -f my-vcn.yaml
kubectl get ocivcn my-vcn
kubectl describe ocivcn my-vcn
```

### Create a Subnet

Once the VCN is active, retrieve its OCID from `status.status.ocid` and reference it in the subnet:

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciSubnet
metadata:
  name: my-subnet
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: my-subnet
  vcnId: ${vcn-ocid}
  cidrBlock: "10.0.1.0/24"
  dnsLabel: mysubnet
  prohibitPublicIpOnVnic: false
```

Apply and check status:

```bash
kubectl apply -f my-subnet.yaml
kubectl get ocisubnet my-subnet
kubectl describe ocisubnet my-subnet
```

## Deletion

When you delete an `OciVcn` or `OciSubnet` resource, the operator will call the OCI API to delete the underlying resource. Delete subnets before deleting their parent VCN.

```bash
kubectl delete ocisubnet my-subnet
kubectl delete ocivcn my-vcn
```

## Binding to Existing Resources

To manage existing OCI networking resources through OSOK without creating new ones, set the `id` field:

```yaml
# Bind to existing VCN
spec:
  id: ocid1.vcn.oc1.<region>.xxx
  compartmentId: ocid1.compartment.oc1..xxx
  displayName: existing-vcn
  cidrBlock: "10.0.0.0/16"

# Bind to existing Subnet
spec:
  id: ocid1.subnet.oc1.<region>.xxx
  compartmentId: ocid1.compartment.oc1..xxx
  displayName: existing-subnet
  vcnId: ocid1.vcn.oc1.<region>.xxx
  cidrBlock: "10.0.1.0/24"
```
