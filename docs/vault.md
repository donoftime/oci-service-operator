# OCI Vault (Key Management)

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI Vault](https://docs.oracle.com/iaas/Content/KeyManagement/Concepts/keyoverview.htm), Oracle Cloud Infrastructure's managed key management service.

Using this operator you can create, bind, update, and delete OCI Vaults and optionally manage a master encryption key within the vault, directly from your Kubernetes cluster using an `OciVault` custom resource.

> **Note:** The existing `pkg/credhelper/vault/` used by OSOK for OCI Vault-backed Kubernetes secrets is a separate concern — it fetches secrets stored *in* a vault. This feature adds vault **resource lifecycle management**: creating and managing the vault itself (and keys within it).

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage Vaults and Keys in your compartment

## OciVault CRD

The `OciVault` CRD maps to an OCI KMS Vault resource and, optionally, a master encryption Key within that vault.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the vault is created |
| `displayName` | string | Yes | User-friendly display name |
| `vaultType` | string | Yes | Vault type: `DEFAULT` or `VIRTUAL_PRIVATE` |
| `id` | string (OCID) | No | Bind to an existing vault instead of creating one |
| `key` | object | No | Optional key to create and manage within the vault |
| `key.displayName` | string | Yes (if key) | Display name for the key |
| `key.keyShape.algorithm` | string | Yes (if key) | Encryption algorithm: `AES`, `RSA`, or `ECDSA` |
| `key.keyShape.length` | integer | Yes (if key) | Key length in bytes (AES: 16/24/32, RSA: 256/384/512, ECDSA: 32/48/66) |
| `key.id` | string (OCID) | No | Bind to an existing key instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned Vault |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

### Connection Secret

When a vault is successfully provisioned, OSOK automatically creates a Kubernetes Secret with the same name as the `OciVault` resource in the same namespace. The secret contains:

| Key | Description |
|-----|-------------|
| `id` | OCID of the vault |
| `displayName` | Display name of the vault |
| `managementEndpoint` | Endpoint URL for key management operations |
| `cryptoEndpoint` | Endpoint URL for cryptographic operations (encrypt/decrypt) |

## Examples

### Create a DEFAULT vault

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciVault
metadata:
  name: my-vault
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: MyVault
  vaultType: DEFAULT
```

### Create a vault with a managed AES-256 key

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciVault
metadata:
  name: my-vault-with-key
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: MyVaultWithKey
  vaultType: DEFAULT
  key:
    displayName: MyAESKey
    keyShape:
      algorithm: AES
      length: 32
```

Apply the resource:

```bash
kubectl apply -f my-vault.yaml
```

Check status:

```bash
kubectl get ocivault my-vault
kubectl describe ocivault my-vault
```

Retrieve the connection details:

```bash
kubectl get secret my-vault -o yaml
```

## Deletion

When you delete an `OciVault` resource, the operator schedules the vault (and any managed key) for deletion via the OCI API. OCI imposes a minimum 7-day grace period before the vault is permanently deleted. The associated connection secret is removed immediately.

```bash
kubectl delete ocivault my-vault
```

## Binding to an Existing Vault

To manage an existing OCI Vault through OSOK without creating a new one, set the `id` field:

```yaml
spec:
  id: ocid1.vault.oc1.<region>.xxx
  compartmentId: ocid1.compartment.oc1..xxx
  displayName: ExistingVault
  vaultType: DEFAULT
```
