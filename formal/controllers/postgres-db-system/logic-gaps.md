# Logic Gaps

- Fixed: non-terminal lifecycle states no longer report `Active`.
- Fixed: bind-by-ID updates no longer depend on a pre-populated `status.ocid`.
- Fixed: delete now waits for not-found before completion instead of clearing immediately after submit.
- Fixed: secret deletion errors now block delete completion instead of being logged and ignored.
- Fixed: `spec.compartmentId` drift is now reconciled in place through OCI's PostgreSQL compartment-move API before supported updates are applied.
- Fixed: PostgreSQL updates now reconcile freeform and defined tag drift in addition to the earlier display-name/description surface.
- Fixed: `spec.shape`, `spec.instanceOcpuCount`, and `spec.instanceMemoryInGBs` drift now fail closed through CEL and reconcile-time validation before any compartment-move or update mutation is submitted.

## Cluster Exercise Findings (2026-03-13)
- Managed PostgreSQL reconciles still skip the update path after resolving an existing DB system by name. `resolveManagedDbSystem` returns the live instance without calling `UpdatePostgresDbSystem`, so managed-spec drift is silently ignored.
- During the `no_reap=true` tag exercise, the CR spec and CR status both looked healthy (`Active` with `spec.freeformTags.no_reap=true`), but OCI still reported the DB system `ACTIVE` without the `no_reap` freeform tag.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
