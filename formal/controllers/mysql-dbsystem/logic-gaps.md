# Logic Gaps

## Fixed
- `FAILED` lifecycle now returns an unsuccessful reconcile result instead of looking successful.
- Retryable states such as `CREATING`, `UPDATING`, and `INACTIVE` now request requeue rather than silently succeeding.
- Delete is no longer a no-op; it now tracks matching OCI delete work requests and only completes after the work request succeeds or OCI already reports the DB system missing.
- Secret cleanup now happens after OCI delete completion is confirmed by work-request success or by OCI reporting the DB system missing.
- Transient non-lifecycle OCI read failures such as throttling and 5xx responses now request controller-managed requeue instead of surfacing as hard reconcile errors.
- Supported MySQL update reconciliation now includes backup policy drift, data-storage growth, hostname label drift, HA toggles, maintenance window drift, and shape drift in addition to the earlier display-name/configuration/tag surface.
- Immutable spec changes now fail closed at the CRD boundary for the audited reject surface, while the service manager rejects the live-visible immutable fields before mutation.

## Cluster Exercise Findings (2026-03-13)
- Managed MySQL reconciles still skip the update path after resolving an existing DB system by name. `resolveManagedDbSystem` returns the live instance without calling `UpdateMySqlDbSystem`, so managed-spec drift is silently ignored.
- During the `no_reap=true` tag exercise, the CR spec and CR status both looked healthy (`Active` with `spec.freeformTags.no_reap=true`), but OCI still reported the DB system `ACTIVE` without the `no_reap` freeform tag.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
