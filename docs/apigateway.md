# OCI API Gateway

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports [OCI API Gateway](https://docs.oracle.com/iaas/Content/APIGateway/home.htm), Oracle Cloud Infrastructure's managed API gateway service.

Using this operator you can create, bind, update, and delete API gateways and deployments directly from your Kubernetes cluster using `ApiGateway` and `ApiGatewayDeployment` custom resources.

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage API Gateway resources in your compartment
- A VCN subnet accessible from your cluster nodes

## ApiGateway CRD

The `ApiGateway` CRD maps to an OCI API Gateway instance.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the gateway is created |
| `endpointType` | string | Yes | `PUBLIC` or `PRIVATE` |
| `subnetId` | string (OCID) | Yes | Subnet for the gateway |
| `displayName` | string | No | User-friendly display name |
| `certificateId` | string (OCID) | No | Certificate for HTTPS |
| `networkSecurityGroupIds` | []string | No | NSG OCIDs associated with the gateway |
| `id` | string (OCID) | No | Bind to an existing gateway instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

The `status.status` field is an `OSOKStatus` containing:

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned API Gateway |
| `conditions` | List of status conditions (Provisioning, Active, Failed, etc.) |
| `createdAt` | Timestamp when the resource was created |

## ApiGatewayDeployment CRD

The `ApiGatewayDeployment` CRD maps to an OCI API Gateway Deployment.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the deployment is created |
| `gatewayId` | string (OCID) | Yes | OCID of the API Gateway to deploy to |
| `pathPrefix` | string | Yes | Path prefix for all routes in this deployment |
| `displayName` | string | No | User-friendly display name |
| `routes` | []ApiGatewayRoute | No | List of API routes |
| `id` | string (OCID) | No | Bind to an existing deployment instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

#### ApiGatewayRoute Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Route path (e.g., `/hello`) |
| `methods` | []string | No | HTTP methods (GET, POST, PUT, DELETE, etc.) |
| `backend` | ApiGatewayRouteBackend | Yes | Backend configuration |

#### ApiGatewayRouteBackend Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | `HTTP_BACKEND`, `ORACLE_FUNCTIONS_BACKEND`, or `STOCK_RESPONSE_BACKEND` |
| `url` | string | For HTTP | Backend URL (for `HTTP_BACKEND`) |
| `functionId` | string | For Functions | Oracle Function OCID (for `ORACLE_FUNCTIONS_BACKEND`) |
| `status` | int | For Stock | HTTP status code (for `STOCK_RESPONSE_BACKEND`) |
| `body` | string | For Stock | Response body (for `STOCK_RESPONSE_BACKEND`) |

## Examples

### Create an API Gateway

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: ApiGateway
metadata:
  name: my-api-gateway
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
  displayName: MyApiGateway
  endpointType: PUBLIC
  subnetId: ocid1.subnet.oc1.iad.XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

### Create an API Gateway Deployment

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: ApiGatewayDeployment
metadata:
  name: my-api-deployment
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
  gatewayId: ocid1.apigateway.oc1.iad.XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
  displayName: MyApiDeployment
  pathPrefix: /v1
  routes:
    - path: /hello
      methods:
        - GET
      backend:
        type: HTTP_BACKEND
        url: https://my-backend.example.com/hello
    - path: /echo
      methods:
        - POST
      backend:
        type: STOCK_RESPONSE_BACKEND
        status: 200
        body: '{"message": "echo"}'
```

### Bind to an Existing Gateway

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: ApiGateway
metadata:
  name: existing-api-gateway
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
  endpointType: PUBLIC
  subnetId: ocid1.subnet.oc1.iad.XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
  id: ocid1.apigateway.oc1.iad.XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

## IAM Policies

The OCI Service Operator requires the following IAM policies to manage API Gateway resources:

```
Allow dynamic-group <osok-dynamic-group> to manage api-gateway-family in compartment <compartment-name>
Allow dynamic-group <osok-dynamic-group> to use virtual-network-family in compartment <compartment-name>
```
