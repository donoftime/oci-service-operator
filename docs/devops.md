# OCI DevOps

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI DevOps](https://docs.oracle.com/iaas/Content/devops/using/home.htm), Oracle Cloud Infrastructure's managed CI/CD pipeline service.

Using this operator you can create, bind, update, and delete DevOps projects directly from your Kubernetes cluster using a `DevopsProject` custom resource.

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage DevOps projects in your compartment
- An OCI Notifications Service (ONS) topic for project notifications

## DevopsProject CRD

The `DevopsProject` CRD maps to an OCI DevOps project, which is the top-level container for CI/CD resources (build pipelines, deploy pipelines, repositories, and environments).

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the project is created |
| `name` | string | Yes | Project name (case-sensitive) |
| `notificationTopicId` | string (OCID) | Yes | OCID of the ONS topic for project notifications |
| `description` | string | No | Human-readable description of the project |
| `id` | string (OCID) | No | Bind to an existing project instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned DevOps project |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

## Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: DevopsProject
metadata:
  name: my-devops-project
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  name: MyDevopsProject
  description: CI/CD project for my application
  notificationTopicId: ocid1.onstopic.oc1.iad.xxx
```

Apply the resource:

```bash
kubectl apply -f my-devops-project.yaml
```

Check status:

```bash
kubectl get devopsproject my-devops-project
kubectl describe devopsproject my-devops-project
```

## Deletion

When you delete a `DevopsProject` resource, the operator will call the OCI API to delete the underlying DevOps project.

```bash
kubectl delete devopsproject my-devops-project
```

> **Note:** Deleting a DevOps project will also delete all associated resources (build pipelines, deploy pipelines, repositories) in OCI. Ensure you no longer need these resources before deleting.

## Binding to an Existing Project

To manage an existing OCI DevOps project through OSOK without creating a new one, set the `id` field:

```yaml
spec:
  id: ocid1.devopsproject.oc1.<region>.xxx
  compartmentId: ocid1.compartment.oc1..xxx
  name: ExistingProject
  notificationTopicId: ocid1.onstopic.oc1.iad.xxx
```

## IAM Policy Requirements

The OCI credentials used by OSOK need the following policies to manage DevOps projects:

```
Allow dynamic-group <osok-dynamic-group> to manage devops-family in compartment <compartment-name>
Allow dynamic-group <osok-dynamic-group> to use ons-topics in compartment <compartment-name>
```
