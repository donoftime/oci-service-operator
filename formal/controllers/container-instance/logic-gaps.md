# Logic Gaps

- Fixed: `CREATING` and `UPDATING` were previously reported as success instead of unsuccessful requeue.
- Fixed: bind-by-ID update paths previously relied on `status.ocid` and could target an empty ID on first reconcile.
- Fixed: delete previously completed immediately after submit instead of waiting for a follow-up `GetContainerInstance` to show deletion.
- Fixed: duplicate-name garbage collection is now modeled as best-effort cleanup that stays within the eligible duplicate set and does not block primary reconcile success.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
