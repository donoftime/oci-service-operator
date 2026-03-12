# Object Storage Bucket Logic Gaps

- Fixed: namespace resolution no longer mutates `spec.namespace` inside reconcile without persisting it.
- Fixed: connection secrets no longer emit the literal `<region>` placeholder in the endpoint URL.
- Fixed: delete only completes once a follow-up `GetBucket` proves the bucket is gone; malformed composite IDs now fail closed.
- Fixed: supported bucket update reconciliation now diffs access/versioning and tag drift against live OCI state instead of treating populated fields as unconditional updates.
- Fixed: `spec.compartmentId` drift is now reconciled in place through `UpdateBucketDetails.CompartmentId`.
- Accepted boundary: the formal model covers the supported bucket update surface (`AccessType`, `Versioning`, and freeform/defined tag drift). Storage tier and other unreconciled bucket attributes remain outside the model because the controller does not update them after creation.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
