# Logic Gaps

## Fixed In This Pass

- Pending `PROVISIONING` and `UPDATING` observations now request another reconcile instead of stalling.
- Delete no longer reports success immediately after submitting the OCI delete request.
- Bind-by-ID update no longer depends on a pre-populated `status.ocid`.
- `GetDrgOcid` now paginates through the full list response before concluding there is no display-name match.
- `UpdateDrg` now reconciles defined tags in addition to display name and freeform tags.
- Managed DRG reconciles now continue using `status.ocid` after create/bind, so supported drift updates do not fall back to create-by-name behavior.
- `spec.compartmentId` drift is now reconciled in place through OCI's DRG compartment-move API before other supported updates are applied.

## Accepted Boundaries

- The current TLA+ model intentionally remains lifecycle-centric; OCI pagination and the supported field-drift reconciliation path are enforced in Go code and property tests rather than this minimal lifecycle spec.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
