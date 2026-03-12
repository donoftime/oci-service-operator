# Logic Gaps

## Fixed
- Non-terminal table states such as `CREATING` and `UPDATING` now requeue instead of reporting success.
- Bind-by-ID updates now target the explicit `spec.tableId` when `status.ocid` is still empty.
- Delete now tracks matching OCI delete work requests and only completes after the work request succeeds or OCI already reports the table missing.
- UpdateTable now diffs DDL, limits, and tags against live OCI state instead of treating populated spec fields as unconditional desired drift.
- Managed table reconciles now continue using `status.ocid` after create/name-resolution, so supported drift updates do not fall back to fresh name lookup.
- `spec.compartmentId` drift is now reconciled in place through `UpdateTableDetails.CompartmentId`.

## Accepted Boundaries
- The CRD still models only the table fields that OSOK exposes directly; OCI-managed schema and replication internals outside that surface remain intentionally out of scope for this controller contract.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
