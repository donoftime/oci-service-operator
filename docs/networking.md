# OCI Networking

## Overview

The OCI Service Operator for Kubernetes (OSOK) supports OCI Networking resources, allowing you to manage Virtual Cloud Networks (VCN) and associated resources directly from your Kubernetes cluster.

Supported resources:
- [OciVcn](#ocivcn-crd) — Virtual Cloud Network
- [OciSubnet](#ocisubnet-crd) — Subnet within a VCN
- [OciInternetGateway](#ociinternetgateway-crd) — Internet connectivity for a VCN
- [OciNatGateway](#ocinatgateway-crd) — Outbound-only internet access
- [OciServiceGateway](#ociservicegateway-crd) — Access to Oracle services without internet
- [OciDrg](#ocidrg-crd) — Dynamic Routing Gateway for hybrid connectivity
- [OciSecurityList](#ocisecuritylist-crd) — Subnet-level firewall rules
- [OciNetworkSecurityGroup](#ocinetworksecuritygroup-crd) — VNIC-level security group
- [OciRouteTable](#ociroutetable-crd) — Routing rules for subnet traffic

## Prerequisites

- OCI Service Operator installed in your cluster
- Appropriate OCI IAM policies to manage networking resources in your compartment
- A compartment OCID where the resources will be created

---

## OciVcn CRD

The `OciVcn` CRD maps to an OCI [Virtual Cloud Network](https://docs.oracle.com/iaas/Content/Network/Tasks/managingVCNs.htm).

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

### Example

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

```bash
kubectl apply -f my-vcn.yaml
kubectl get ocivcn my-vcn
kubectl describe ocivcn my-vcn
```

---

## OciSubnet CRD

The `OciSubnet` CRD maps to an OCI [Subnet](https://docs.oracle.com/iaas/Content/Network/Tasks/managingsubnets.htm) within a VCN.

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

### Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciSubnet
metadata:
  name: my-subnet
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: my-subnet
  vcnId: ocid1.vcn.oc1.phx.aaaaaaaaxxx
  cidrBlock: "10.0.1.0/24"
  dnsLabel: mysubnet
  prohibitPublicIpOnVnic: false
```

```bash
kubectl apply -f my-subnet.yaml
kubectl get ocisubnet my-subnet
kubectl describe ocisubnet my-subnet
```

---

## OciInternetGateway CRD

The `OciInternetGateway` CRD manages an [OCI Internet Gateway](https://docs.oracle.com/iaas/Content/Network/Tasks/managingIGs.htm), which provides two-way internet connectivity for resources in a VCN. Attach a route table rule pointing `0.0.0.0/0` at the Internet Gateway to make a subnet public.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the gateway is created |
| `vcnId` | string (OCID) | Yes | OCID of the VCN that contains this gateway |
| `displayName` | string | Yes | User-friendly display name |
| `isEnabled` | bool | No | Whether the gateway is enabled (default: true) |
| `id` | string (OCID) | No | Bind to an existing Internet Gateway instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned Internet Gateway |
| `conditions` | List of status conditions |
| `createdAt` | Timestamp when the resource was created |

### Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciInternetGateway
metadata:
  name: my-igw
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  vcnId: ocid1.vcn.oc1.phx.aaaaaaaaxxx
  displayName: my-igw
  isEnabled: true
```

```bash
kubectl apply -f my-igw.yaml
kubectl get ociinternetgateway my-igw
kubectl describe ociinternetgateway my-igw
```

---

## OciNatGateway CRD

The `OciNatGateway` CRD manages an [OCI NAT Gateway](https://docs.oracle.com/iaas/Content/Network/Tasks/NATgateway.htm), which provides outbound-only internet access for resources in private subnets. Instances can initiate outbound connections but cannot receive inbound connections from the internet.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the gateway is created |
| `vcnId` | string (OCID) | Yes | OCID of the VCN that contains this gateway |
| `displayName` | string | Yes | User-friendly display name |
| `blockTraffic` | bool | No | When true, blocks all traffic through the NAT Gateway (default: false) |
| `id` | string (OCID) | No | Bind to an existing NAT Gateway instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Status Fields

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned NAT Gateway |
| `conditions` | List of status conditions |
| `createdAt` | Timestamp when the resource was created |

### Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciNatGateway
metadata:
  name: my-natgw
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  vcnId: ocid1.vcn.oc1.phx.aaaaaaaaxxx
  displayName: my-natgw
  blockTraffic: false
```

```bash
kubectl apply -f my-natgw.yaml
kubectl get ocinatgateway my-natgw
kubectl describe ocinatgateway my-natgw
```

---

## OciServiceGateway CRD

The `OciServiceGateway` CRD manages an [OCI Service Gateway](https://docs.oracle.com/iaas/Content/Network/Tasks/servicegateway.htm), which enables private access to Oracle services (such as Object Storage, Streaming, and others) from within a VCN without traffic leaving Oracle's network or requiring internet access.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the gateway is created |
| `vcnId` | string (OCID) | Yes | OCID of the VCN that contains this gateway |
| `displayName` | string | Yes | User-friendly display name |
| `services` | []string | Yes | List of OCI service OCIDs to enable on this gateway |
| `id` | string (OCID) | No | Bind to an existing Service Gateway instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Notes

The `services` field accepts OCI service label OCIDs. Use the OCI Console or CLI to discover the service OCIDs for your region. Common values include the "All <region> Services in Oracle Services Network" bundle OCID. Service OCIDs are region-specific.

### Status Fields

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned Service Gateway |
| `conditions` | List of status conditions |
| `createdAt` | Timestamp when the resource was created |

### Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciServiceGateway
metadata:
  name: my-svcgw
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  vcnId: ocid1.vcn.oc1.phx.aaaaaaaaxxx
  displayName: my-svcgw
  services:
    - ocid1.service.oc1.phx.aaaaaaaa...  # "All PHX Services in Oracle Services Network"
```

```bash
kubectl apply -f my-svcgw.yaml
kubectl get ociservicegateway my-svcgw
kubectl describe ociservicegateway my-svcgw
```

---

## OciDrg CRD

The `OciDrg` CRD manages an [OCI Dynamic Routing Gateway (DRG)](https://docs.oracle.com/iaas/Content/Network/Tasks/managingDRGs.htm), which is a virtual router that connects a VCN to on-premises networks via FastConnect or VPN, or to other VCNs via VCN peering. Unlike other networking resources, a DRG is created at the compartment level, not within a specific VCN.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the DRG is created |
| `displayName` | string | Yes | User-friendly display name |
| `id` | string (OCID) | No | Bind to an existing DRG instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Notes

The DRG is a compartment-level resource and does not have a `vcnId` field. Attach it to a VCN using the OCI Console or API after creation. The DRG OCID from `status.status.ocid` can then be used as a route target in an `OciRouteTable`.

### Status Fields

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned DRG |
| `conditions` | List of status conditions |
| `createdAt` | Timestamp when the resource was created |

### Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciDrg
metadata:
  name: my-drg
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  displayName: my-drg
```

```bash
kubectl apply -f my-drg.yaml
kubectl get ocidrg my-drg
kubectl describe ocidrg my-drg
```

---

## OciSecurityList CRD

The `OciSecurityList` CRD manages an [OCI Security List](https://docs.oracle.com/iaas/Content/Network/Concepts/securitylists.htm), which is a set of firewall rules applied at the subnet level. All VNICs in a subnet share the same security lists. Rules can be stateful (connection tracking) or stateless (evaluated independently per packet).

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the security list is created |
| `vcnId` | string (OCID) | Yes | OCID of the VCN that contains this security list |
| `displayName` | string | Yes | User-friendly display name |
| `ingressSecurityRules` | []IngressSecurityRule | No | Ingress (inbound) firewall rules |
| `egressSecurityRules` | []EgressSecurityRule | No | Egress (outbound) firewall rules |
| `id` | string (OCID) | No | Bind to an existing Security List instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

#### IngressSecurityRule Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `protocol` | string | Yes | IP protocol number: `"all"`, `"6"` (TCP), `"17"` (UDP), `"1"` (ICMP) |
| `source` | string | Yes | Source CIDR block (e.g. `"0.0.0.0/0"`) |
| `isStateless` | bool | No | When true, rule is stateless (default: false, stateful) |
| `description` | string | No | Human-readable description |
| `tcpOptions` | TcpOptions | No | TCP port range filter (only for protocol `"6"`) |
| `udpOptions` | UdpOptions | No | UDP port range filter (only for protocol `"17"`) |

#### EgressSecurityRule Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `protocol` | string | Yes | IP protocol number: `"all"`, `"6"` (TCP), `"17"` (UDP), `"1"` (ICMP) |
| `destination` | string | Yes | Destination CIDR block (e.g. `"0.0.0.0/0"`) or service CIDR label |
| `destinationType` | string | No | `"CIDR_BLOCK"` (default) or `"SERVICE_CIDR_BLOCK"` |
| `isStateless` | bool | No | When true, rule is stateless (default: false, stateful) |
| `description` | string | No | Human-readable description |
| `tcpOptions` | TcpOptions | No | TCP port range filter (only for protocol `"6"`) |
| `udpOptions` | UdpOptions | No | UDP port range filter (only for protocol `"17"`) |

#### TcpOptions / UdpOptions Fields

| Field | Type | Description |
|-------|------|-------------|
| `destinationPortRange` | PortRange | Destination port range (`min`, `max`) |
| `sourcePortRange` | PortRange | Source port range (`min`, `max`) |

### Reconciliation Behavior

Security rules are reconciled on every controller cycle. If you update `ingressSecurityRules` or `egressSecurityRules` in the spec, the controller applies the full set of rules to OCI on the next reconcile — replacing any previously configured rules. This ensures the OCI Security List always reflects the spec exactly.

### Status Fields

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned Security List |
| `conditions` | List of status conditions |
| `createdAt` | Timestamp when the resource was created |

### Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciSecurityList
metadata:
  name: my-seclist
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  vcnId: ocid1.vcn.oc1.phx.aaaaaaaaxxx
  displayName: my-seclist
  ingressSecurityRules:
    - protocol: "6"     # TCP
      source: "0.0.0.0/0"
      description: "Allow HTTPS"
      tcpOptions:
        destinationPortRange:
          min: 443
          max: 443
    - protocol: "all"
      source: "10.0.0.0/16"
      description: "Allow all VCN-internal traffic"
  egressSecurityRules:
    - protocol: "all"
      destination: "0.0.0.0/0"
      description: "Allow all outbound traffic"
```

```bash
kubectl apply -f my-seclist.yaml
kubectl get ocisecuritylist my-seclist
kubectl describe ocisecuritylist my-seclist
```

---

## OciNetworkSecurityGroup CRD

The `OciNetworkSecurityGroup` CRD manages an [OCI Network Security Group (NSG)](https://docs.oracle.com/iaas/Content/Network/Concepts/networksecuritygroups.htm), which is a VNIC-level security construct. Unlike Security Lists (which apply to all VNICs in a subnet), NSG rules are applied to individual VNICs that are explicitly added to the group. NSGs are the recommended approach for fine-grained security policy.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the NSG is created |
| `vcnId` | string (OCID) | Yes | OCID of the VCN that contains this NSG |
| `displayName` | string | Yes | User-friendly display name |
| `id` | string (OCID) | No | Bind to an existing NSG instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

### Notes

After creating an NSG with OSOK, add security rules and associate VNICs using the OCI Console or API. The NSG OCID from `status.status.ocid` can be referenced in compute instance or container instance specs to place VNICs into the group.

### Status Fields

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned NSG |
| `conditions` | List of status conditions |
| `createdAt` | Timestamp when the resource was created |

### Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciNetworkSecurityGroup
metadata:
  name: my-nsg
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  vcnId: ocid1.vcn.oc1.phx.aaaaaaaaxxx
  displayName: my-nsg
```

```bash
kubectl apply -f my-nsg.yaml
kubectl get ocinetworksecuritygroup my-nsg
kubectl describe ocinetworksecuritygroup my-nsg
```

---

## OciRouteTable CRD

The `OciRouteTable` CRD manages an [OCI Route Table](https://docs.oracle.com/iaas/Content/Network/Tasks/managingroutetables.htm), which contains routing rules that control where traffic is directed when leaving a subnet. Associate a route table with a subnet using the `routeTableId` field on `OciSubnet`.

### Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `compartmentId` | string (OCID) | Yes | Compartment where the route table is created |
| `vcnId` | string (OCID) | Yes | OCID of the VCN that contains this route table |
| `displayName` | string | Yes | User-friendly display name |
| `routeRules` | []RouteRule | No | List of routing rules |
| `id` | string (OCID) | No | Bind to an existing Route Table instead of creating one |
| `freeformTags` | map | No | OCI freeform tags |
| `definedTags` | map | No | OCI defined tags |

#### RouteRule Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `networkEntityId` | string (OCID) | Yes | OCID of the gateway to route traffic to (IGW, NAT GW, etc.) |
| `destination` | string | Yes | Destination CIDR block (e.g. `"0.0.0.0/0"`) or service CIDR label |
| `destinationType` | string | No | `"CIDR_BLOCK"` (default) or `"SERVICE_CIDR_BLOCK"` |
| `description` | string | No | Human-readable description |

### Reconciliation Behavior

Route rules are reconciled on every controller cycle. If you update `routeRules` in the spec, the controller applies the full set of rules to OCI on the next reconcile — replacing any previously configured rules. This ensures the OCI Route Table always reflects the spec exactly.

### Status Fields

| Field | Description |
|-------|-------------|
| `ocid` | OCID of the provisioned Route Table |
| `conditions` | List of status conditions |
| `createdAt` | Timestamp when the resource was created |

### Example

```yaml
apiVersion: oci.oracle.com/v1beta1
kind: OciRouteTable
metadata:
  name: my-rt
  namespace: default
spec:
  compartmentId: ocid1.compartment.oc1..aaaaaaaaxxx
  vcnId: ocid1.vcn.oc1.phx.aaaaaaaaxxx
  displayName: my-rt
  routeRules:
    - destination: "0.0.0.0/0"
      networkEntityId: ocid1.internetgateway.oc1.phx.aaaaaaaaxxx
      destinationType: CIDR_BLOCK
      description: "Default route to Internet Gateway"
```

```bash
kubectl apply -f my-rt.yaml
kubectl get ociroutetable my-rt
kubectl describe ociroutetable my-rt
```

---

## Deletion

When you delete networking resources, do so in reverse dependency order. For example, delete subnets before route tables, and route tables before gateways and VCNs.

```bash
kubectl delete ocisubnet my-subnet
kubectl delete ociroutetable my-rt
kubectl delete ocisecuritylist my-seclist
kubectl delete ocinetworksecuritygroup my-nsg
kubectl delete ociinternetgateway my-igw
kubectl delete ocinatgateway my-natgw
kubectl delete ociservicegateway my-svcgw
kubectl delete ocisubnet my-subnet   # (if not already deleted)
kubectl delete ocivcn my-vcn
```

## Binding to Existing Resources

All networking CRDs support binding to existing OCI resources by setting the `id` field:

```yaml
spec:
  id: ocid1.internetgateway.oc1.<region>.xxx
  compartmentId: ocid1.compartment.oc1..xxx
  vcnId: ocid1.vcn.oc1.<region>.xxx
  displayName: existing-igw
```

When `id` is set, the operator adopts the existing resource instead of creating a new one. The resource will be deleted from OCI when the Kubernetes object is deleted.
