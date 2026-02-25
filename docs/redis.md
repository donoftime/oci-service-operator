# OCI Cache with Redis Service

The OCI Service Operator for Kubernetes (OSOK) supports provisioning and managing [OCI Cache with Redis](https://docs.oracle.com/en-us/iaas/Content/redis/home.htm) clusters. A Redis cluster is a memory-based storage solution that supports read/write operations from applications running in OKE or other Kubernetes clusters.

## Prerequisites

- An OCI tenancy with permissions to create Redis clusters
- A subnet OCID in the same VCN as your Kubernetes cluster
- OSOK deployed and configured with appropriate OCI credentials

## Resource: RedisCluster

The `RedisCluster` CRD manages OCI Cache Redis clusters. The operator handles:

- **Create**: Provisions a new Redis cluster with the specified configuration
- **Bind**: Attaches to an existing Redis cluster by OCID (set `spec.Id`)
- **Delete**: Deletes the Redis cluster when the CR is removed (respects finalizers)
- **Status**: Stores the cluster OCID, primary/replicas endpoints in Kubernetes secrets

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | OCID of the OCI compartment |
| `displayName` | string | Yes | Human-readable name for the cluster |
| `nodeCount` | integer | Yes | Number of nodes (minimum: 1) |
| `nodeMemoryInGBs` | float | Yes | Memory per node in gigabytes |
| `softwareVersion` | string | Yes | Redis version. Allowed: `V7_0_5` |
| `subnetId` | string (OCID) | Yes | OCID of the subnet for the cluster |
| `Id` | string (OCID) | No | OCID of an existing cluster to bind to |
| `freeformTags` | map | No | Freeform tags for the resource |
| `definedTags` | map | No | Defined tags for the resource |

## Status Fields

The operator populates `status.status` with standard OSOK fields:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned Redis cluster |
| `conditions` | Lifecycle conditions (Provisioning, Active, Failed, Updating, Terminating) |
| `createdAt` | Timestamp when the cluster was provisioned |

## Connection Secret

When a Redis cluster becomes Active, OSOK creates a Kubernetes Secret in the same namespace with the name matching the CR name. The secret contains:

| Key | Description |
|-----|-------------|
| `primaryFqdn` | FQDN of the primary node API endpoint |
| `primaryEndpointIpAddress` | Private IP of the primary node endpoint |
| `replicasFqdn` | FQDN of the replica nodes API endpoint |
| `replicasEndpointIpAddress` | Private IP of the replica nodes endpoint |

## Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: RedisCluster
metadata:
  name: my-redis-cluster
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaXXXXXXXXXXXXXXXX
  displayName: MyRedisCluster
  nodeCount: 3
  nodeMemoryInGBs: 4.0
  softwareVersion: V7_0_5
  subnetId: ocid1.subnet.oc1.iad.aaaaaaaaXXXXXXXXXXXXXXXX
```

Apply the resource:

```bash
kubectl apply -f oci_v1beta1_rediscluster.yaml
```

Check status:

```bash
kubectl get rediscluster my-redis-cluster
kubectl describe rediscluster my-redis-cluster
```

Read the connection secret:

```bash
kubectl get secret my-redis-cluster -o jsonpath='{.data.primaryFqdn}' | base64 -d
```

## Binding to an Existing Redis Cluster

To bind OSOK management to an already-provisioned Redis cluster, specify the `Id` field:

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: RedisCluster
metadata:
  name: existing-redis-cluster
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaXXXXXXXXXXXXXXXX
  displayName: ExistingCluster
  nodeCount: 3
  nodeMemoryInGBs: 4.0
  softwareVersion: V7_0_5
  subnetId: ocid1.subnet.oc1.iad.aaaaaaaaXXXXXXXXXXXXXXXX
  Id: ocid1.rediscluster.oc1.iad.XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

## RBAC

The operator requires the following RBAC permissions:

- `redisclusters`: get, list, watch, create, update, patch, delete
- `redisclusters/status`: get, update, patch
- `redisclusters/finalizers`: update
- `secrets`: get, list, watch, create, update, patch, delete (for connection secrets)
- `events`: get, list, watch, create, update, patch, delete

Editor and viewer ClusterRoles are provided in `config/rbac/rediscluster_editor_role.yaml` and `config/rbac/rediscluster_viewer_role.yaml`.
