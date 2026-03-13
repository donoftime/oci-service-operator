# Logic Gaps

- Fixed: `CREATING` and `UPDATING` were previously reported as success instead of unsuccessful requeue.
- Fixed: bind-by-ID update paths previously relied on `status.ocid` and could target an empty ID on first reconcile.
- Fixed: delete previously completed immediately after submit instead of waiting for a follow-up `GetContainerInstance` to show deletion.
- Fixed: duplicate-name garbage collection is now modeled as best-effort cleanup that stays within the eligible duplicate set and does not block primary reconcile success.

## Cluster Exercise Findings (2026-03-13)
- The create path still emits a false error log, `key and value must be string`, before the OCI create request succeeds. This comes from the structured logger call in `CreateContainerInstance` passing a non-string value to `DebugLog`, so successful creates look like controller errors in the operator logs.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
