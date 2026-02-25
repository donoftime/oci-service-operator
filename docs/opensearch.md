# OCI Search Service with OpenSearch

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI Search Service with OpenSearch](https://docs.oracle.com/iaas/Content/search-opensearch/home.htm), Oracle Cloud Infrastructure's managed OpenSearch cluster service.

Using this operator you can create, bind, update, and delete OpenSearch clusters directly from your Kubernetes cluster using an `OpenSearchCluster` custom resource.

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage OpenSearch clusters in your compartment
- A VCN and subnet accessible from your cluster nodes

## OpenSearchCluster CRD

The `OpenSearchCluster` CRD maps to an OCI OpenSearch cluster.

### Spec Fields

#### Core Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the cluster is created |
| `displayName` | string | Yes | User-friendly display name |
| `softwareVersion` | string | Yes | OpenSearch version (e.g. `2.11.0`) |
| `id` | string (OCID) | No | Bind to an existing cluster instead of creating one |

#### Master Node Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `masterNodeCount` | integer | Yes | Number of master nodes (minimum 1) |
| `masterNodeHostType` | string | Yes | Instance type: `FLEX` or `BM` |
| `masterNodeHostOcpuCount` | integer | Yes | OCPUs per master node |
| `masterNodeHostMemoryGB` | integer | Yes | Memory in GB per master node |
| `masterNodeHostBareMetalShape` | string | No | Bare metal shape (when hostType is `BM`) |

#### Data Node Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `dataNodeCount` | integer | Yes | Number of data nodes (minimum 1) |
| `dataNodeHostType` | string | Yes | Instance type: `FLEX`, `BM`, or `DENSE_IO` |
| `dataNodeHostOcpuCount` | integer | Yes | OCPUs per data node |
| `dataNodeHostMemoryGB` | integer | Yes | Memory in GB per data node |
| `dataNodeStorageGB` | integer | Yes | Storage in GB per data node |
| `dataNodeHostBareMetalShape` | string | No | Bare metal shape (when hostType is `BM`) |

#### OpenSearch Dashboard Node Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `opendashboardNodeCount` | integer | Yes | Number of dashboard nodes (minimum 1) |
| `opendashboardNodeHostOcpuCount` | integer | Yes | OCPUs for dashboard nodes |
| `opendashboardNodeHostMemoryGB` | integer | Yes | Memory in GB for dashboard nodes |

#### Networking

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `vcnId` | string (OCID) | Yes | VCN OCID |
| `subnetId` | string (OCID) | Yes | Subnet OCID |
| `vcnCompartmentId` | string (OCID) | Yes | Compartment of the VCN |
| `subnetCompartmentId` | string (OCID) | Yes | Compartment of the subnet |

#### Security

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `securityMode` | string | No | `DISABLED`, `PERMISSIVE`, or `ENFORCING` |
| `securityMasterUserName` | string | No | Master user name for security config |
| `securityMasterUserPasswordHash` | string | No | Password hash for the master user |

#### Tags

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned OpenSearch cluster |
| `conditions` | List of status conditions (Provisioning, Active, Updating, Failed) |
| `createdAt` | Timestamp when the resource was created |

## Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OpenSearchCluster
metadata:
  name: my-opensearch-cluster
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaXXXXX
  displayName: MyOpenSearchCluster
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
  opendashboardNodeHostMemoryGB: 8

  vcnId: ocid1.vcn.oc1.us-ashburn-1.aaaaaaaXXXXX
  subnetId: ocid1.subnet.oc1.us-ashburn-1.aaaaaaaXXXXX
  vcnCompartmentId: ocid1.compartment.oc1..aaaaaaaXXXXX
  subnetCompartmentId: ocid1.compartment.oc1..aaaaaaaXXXXX

  securityMode: PERMISSIVE
```

## Bind to Existing Cluster

To adopt an existing OpenSearch cluster without creating a new one, specify its OCID in the `id` field:

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OpenSearchCluster
metadata:
  name: existing-cluster
spec:
  id: ocid1.opensearchcluster.oc1.us-ashburn-1.aaaaaaaXXXXX
  compartmentId: ocid1.compartment.oc1..aaaaaaaXXXXX
  displayName: ExistingCluster
```

## Lifecycle States

The operator maps OCI lifecycle states to OSOK conditions:

| OCI State | OSOK Condition |
|-----------|----------------|
| CREATING | Provisioning |
| ACTIVE | Active |
| UPDATING | Updating |
| FAILED | Failed |
| DELETING | Terminating |
