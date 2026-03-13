# Logic Gaps

- Fixed: non-terminal lifecycle states no longer report `Active`.
- Fixed: bind-by-ID updates no longer depend on `status.ocid` being populated on the first reconcile.
- Fixed: delete now waits for `DELETED` or not-found before completion instead of returning done immediately.
- Fixed: secret deletion errors now block delete completion instead of being logged and ignored.
- Fixed: managed Redis reconciles now continue using `status.ocid` after create/name-resolution, so supported drift updates do not fall back to fresh name lookup.
- Fixed: `spec.compartmentId` drift is now reconciled in place through OCI's Redis cluster compartment-move API before other supported updates are applied.
- Fixed: Redis updates now reconcile freeform and defined tag drift, and subnet changes are rejected at the CRD boundary before reconcile.
- Fixed: `spec.softwareVersion` drift now fails closed through CEL and reconcile-time validation before any compartment-move or update mutation is submitted.

## Cluster Exercise Findings (2026-03-13)
- During `OSOKPlatform` teardown, the Redis CR deleted successfully before the VCN could be deleted, and the VCN later failed delete with `409 IncorrectState` because a security list was still in use.
- The remaining provider-side VCN dependency was observed as a security list named `redis-security-list`. Redis only models a subnet in the CR surface, so this appears to be a provider-managed side artifact that outlives the Redis CR.
- The Redis delete path may therefore be declaring success before OCI has fully cleaned up Redis-managed networking dependencies needed for downstream VCN deletion.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
