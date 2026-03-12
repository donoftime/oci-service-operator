# OCI Queue Logic Gaps

- Fixed: async creation and non-terminal queue states now explicitly request requeue.
- Fixed: bind-by-ID updates no longer depend on a pre-populated `status.ocid`.
- Fixed: delete only completes once `GetQueue` confirms deletion or not-found.
- Fixed: supported queue update reconciliation now includes tag drift in addition to the existing display name and timeout/visibility settings.
- Fixed: managed queue reconciles now continue using `status.ocid` after create/name-resolution, so supported drift updates do not fall back to fresh name lookup.
- Fixed: `spec.compartmentId` drift is now reconciled in place through OCI's queue compartment-move API before other supported updates are applied.
- Fixed: supported queue update reconciliation now includes non-empty `spec.customEncryptionKeyId` drift.
- Fixed: unsupported `spec.retentionInSeconds` drift is now rejected before any compartment-move or update mutation is submitted.
- Accepted boundary: the formal model covers the queue fields OSOK reconciles after create (`DisplayName`, visibility, timeout, dead-letter delivery count, freeform/defined tags, and non-empty custom-encryption-key drift). `retentionInSeconds` remains outside the in-place drift model because the controller rejects it, and empty `customEncryptionKeyId` remains non-destructive because the CRD does not distinguish unset from explicit key removal.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
