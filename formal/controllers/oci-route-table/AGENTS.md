# OciRouteTable

- Source of truth: `spec.tla` and `spec.cfg`
- Shared contract: `../../shared/BaseReconcilerContract.tla`
- Diagram source: `diagrams/lifecycle.puml`
- Go enforcement: `pkg/servicemanager/networking/networking_properties_test.go`
- Controller-specific notes: `logic-gaps.md`

## Verified Properties

- `TypeInvariant`
- `SuccessRequiresActiveInvariant`
- `RetryableRequiresRequeueInvariant`
- `DeleteRequiresResourceGoneInvariant`
- `SecretRequiresUsableStateInvariant`

## Networking Notes

- Fixed in Go: retryable OCI lifecycle states now return `ShouldRequeue=true`.
- Fixed in Go: delete is only complete after a follow-up `GetRouteTable` confirms the resource is gone.
- Fixed in Go: explicit-ID bind/update uses the spec OCID even when `status.ocid` starts empty.
- Related Go tests:
  - `TestPropertyNetworkingPendingStatesRequestRequeue`
  - `TestPropertyNetworkingBindByIDUsesExplicitSpecIDWhenStatusIsEmpty`
  - `TestPropertyNetworkingDeleteWaitsForConfirmedDisappearance`
