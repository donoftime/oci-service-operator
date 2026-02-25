# OCI Cache with Redis

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI Cache with Redis](https://docs.oracle.com/iaas/Content/redis/home.htm), Oracle Cloud Infrastructure's managed Redis cluster service.

Using this operator you can create, bind, update, and delete Redis clusters directly from your Kubernetes cluster using a `RedisCluster` custom resource.

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage Redis clusters in your compartment
- A VCN subnet accessible from your cluster nodes

## RedisCluster CRD

The `RedisCluster` CRD maps to an OCI Cache Redis cluster.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the cluster is created |
| `displayName` | string | Yes | User-friendly display name |
| `nodeCount` | integer | Yes | Number of nodes (minimum 1) |
| `nodeMemoryInGBs` | float | Yes | Memory per node in gigabytes |
| `softwareVersion` | string | Yes | Redis version (e.g. `V7_0_5`) |
| `subnetId` | string (OCID) | Yes | Subnet for the cluster's endpoints |
| `id` | string (OCID) | No | Bind to an existing cluster instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned Redis cluster |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

### Connection Secret

When a cluster is successfully provisioned, OSOK automatically creates a Kubernetes Secret with the same name as the `RedisCluster` resource in the same namespace. The secret contains:

| Key | Description |
|-----|-------------|
| `primaryFqdn` | FQDN of the primary endpoint |
| `primaryEndpointIpAddress` | Private IP of the primary endpoint |
| `replicasFqdn` | FQDN of the replica endpoints |
| `replicasEndpointIpAddress` | Private IP of the replica endpoints |

## Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: RedisCluster
metadata:
  name: my-redis-cluster
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: MyRedisCluster
  nodeCount: 3
  nodeMemoryInGBs: 16
  softwareVersion: V7_0_5
  subnetId: ocid1.subnet.oc1.iad.xxx
```

Apply the resource:

```bash
kubectl apply -f my-redis-cluster.yaml
```

Check status:

```bash
kubectl get rediscluster my-redis-cluster
kubectl describe rediscluster my-redis-cluster
```

Retrieve the connection details:

```bash
kubectl get secret my-redis-cluster -o yaml
```

## Deletion

When you delete a `RedisCluster` resource, the operator will call the OCI API to delete the underlying Redis cluster and remove the associated connection secret.

```bash
kubectl delete rediscluster my-redis-cluster
```

## Binding to an Existing Cluster

To manage an existing OCI Redis cluster through OSOK without creating a new one, set the `id` field:

```yaml
spec:
  id: ocid1.redis.oc1.<region>.xxx
  compartmentId: ocid1.compartment.oc1..xxx
  displayName: ExistingCluster
  nodeCount: 3
  nodeMemoryInGBs: 16
  softwareVersion: V7_0_5
  subnetId: ocid1.subnet.oc1.iad.xxx
```
