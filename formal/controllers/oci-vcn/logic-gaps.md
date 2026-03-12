# Logic Gaps

## Fixed In This Pass

- Pending `PROVISIONING` and `UPDATING` observations now request another reconcile instead of stalling.
- Delete no longer reports success immediately after submitting the OCI delete request.
- Bind-by-ID update no longer depends on a pre-populated `status.ocid`.
- `UpdateVcn` now reconciles defined tags in addition to display name and freeform tags.

## Accepted Boundaries

- The current TLA+ model intentionally remains lifecycle-centric; the supported field-drift reconciliation path is enforced in Go code and property tests rather than this minimal lifecycle spec.
