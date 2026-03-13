# Logic Gaps

- Fixed: `PROVISIONING`, `STARTING`, and `STOPPING` no longer report terminal success.
- Fixed: bind-by-ID update paths now target the explicit spec OCID even when `status.ocid` is unset.
- Fixed: delete no longer completes immediately after terminate submission; it waits for `TERMINATED` or not-found.
- Fixed: bound-instance reconciles now diff and apply freeform/defined tag drift instead of limiting updates to display name only.
- Fixed: managed compute reconciles now continue using `status.ocid` after create/name-resolution, so supported drift updates do not fall back to fresh name lookup.
- Fixed: `spec.compartmentId` drift is now reconciled in place through OCI's instance compartment-move API before other supported updates are applied.
- Fixed: `spec.imageId` drift now fails closed through CEL and reconcile-time validation before any compartment-move or update mutation is submitted.
- Accepted boundary: the formal model intentionally stops at the fields OSOK actually reconciles on bound instances: display name, shape, shape config, and freeform/defined tags. Image, subnet, and instance-option drift remain outside the model because the controller rejects or does not update those fields after bind.

## Cluster Exercise Findings (2026-03-13)
- The create path still emits a false error log, `key and value must be string`, before the OCI launch request succeeds. This comes from the structured logger call in `LaunchInstance` passing a non-string value to `DebugLog`, so successful creates look like controller errors in the operator logs.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
