# Functions Application Logic Gaps

- Fixed: bind-by-ID updates no longer rely on a blank `status.ocid`; they use the explicit spec ID.
- Fixed: `CREATING`, `UPDATING`, and `DELETING` are treated as retryable, not successful.
- Fixed: delete only completes once a follow-up `GetApplication` shows the resource is deleted or not found.
- Fixed: update reconciliation now diffs config, NSG IDs, syslog URL, and tags against live OCI state, and the name-resolved existing-application path applies those updates through the resolved OCID.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
