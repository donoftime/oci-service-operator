# OCI Functions Service

The OCI Service Operator for Kubernetes (OSOK) supports managing OCI Functions applications and functions as Kubernetes custom resources.

## Overview

OCI Functions is a serverless platform that lets you create, run, and scale functions without managing infrastructure. OSOK provides two CRDs:

- **FunctionsApplication** — an application container that holds functions and shared configuration
- **FunctionsFunction** — an individual serverless function (Docker image + config) within an application

## Prerequisites

- OCI Functions service enabled in the tenancy
- A VCN with at least one subnet configured for functions
- An OCI Container Registry (OCIR) repository with your function image

## FunctionsApplication

### Spec Fields

| Field | Required | Description |
|---|---|---|
| `compartmentId` | Yes | OCID of the compartment to create the application in |
| `displayName` | Yes | Display name of the application (unique within compartment) |
| `subnetIds` | Yes | List of subnet OCIDs in which to run functions |
| `id` | No | Bind to an existing application by OCID |
| `config` | No | Key-value configuration passed to all functions as env vars |
| `networkSecurityGroupIds` | No | List of NSG OCIDs |
| `syslogUrl` | No | Syslog URL for function logs |
| `shape` | No | `GENERIC_X86`, `GENERIC_ARM`, or `GENERIC_X86_ARM` |
| `freeformTags` | No | OCI freeform tags |
| `definedTags` | No | OCI defined tags |

### Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: FunctionsApplication
metadata:
  name: my-app
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaa
  displayName: my-functions-app
  subnetIds:
    - ocid1.subnet.oc1..aaaa
  config:
    ENV: production
  shape: GENERIC_X86
```

### Status

```bash
kubectl get functionsapplication my-app
kubectl describe functionsapplication my-app
```

## FunctionsFunction

### Spec Fields

| Field | Required | Description |
|---|---|---|
| `applicationId` | Yes | OCID of the FunctionsApplication this function belongs to |
| `displayName` | Yes | Display name of the function (unique within application) |
| `image` | Yes | Qualified Docker image name in OCIR |
| `memoryInMBs` | Yes | Maximum memory for the function in MiB (minimum 128) |
| `id` | No | Bind to an existing function by OCID |
| `timeoutInSeconds` | No | Execution timeout in seconds |
| `config` | No | Key-value configuration (overrides application config) |
| `freeformTags` | No | OCI freeform tags |
| `definedTags` | No | OCI defined tags |

### Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: FunctionsFunction
metadata:
  name: my-function
  namespace: default
spec:
  applicationId: ocid1.fnapp.oc1..aaaa
  displayName: my-function
  image: phx.ocir.io/mytenancy/myrepo/my-function:0.0.1
  memoryInMBs: 256
  timeoutInSeconds: 30
  config:
    LOG_LEVEL: info
```

### Invoke Endpoint Secret

When a FunctionsFunction is created, OSOK stores the invoke endpoint and function OCID in a Kubernetes secret with the same name as the resource:

```bash
kubectl get secret my-function -o yaml
```

Keys:
- `invokeEndpoint` — HTTPS URL to invoke the function
- `functionId` — OCID of the function

### Status

```bash
kubectl get functionsfunction my-function
kubectl describe functionsfunction my-function
```

## Binding to Existing Resources

To manage an existing OCI resource, specify its OCID in the `id` field:

```yaml
spec:
  id: ocid1.fnapp.oc1..existing
  compartmentId: ocid1.compartment.oc1..aaaa
  displayName: existing-app
  subnetIds:
    - ocid1.subnet.oc1..aaaa
```

## Deletion

When a FunctionsApplication or FunctionsFunction resource is deleted from Kubernetes, OSOK deletes the corresponding OCI resource. Functions must be deleted before their parent application.
