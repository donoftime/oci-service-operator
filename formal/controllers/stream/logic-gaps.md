# Stream Logic Gaps

- Fixed: existing-by-name updates now target the resolved stream ID instead of a blank spec ID.
- Fixed: `FAILED` and `DELETED` streams no longer report terminal success.
- Fixed: `CREATING`, `UPDATING`, and `DELETING` now requeue instead of silently stalling.
- Fixed: delete resolves IDs from spec, status, or name lookup and only completes once the stream is gone.
- Fixed: secret generation now fails safely when `MessagesEndpoint` is missing instead of panicking.
- Fixed: managed stream reconciles now continue using `status.ocid` after create/name-resolution, so supported drift updates do not fall back to fresh name lookup.
- Fixed: `spec.compartmentId` drift is now reconciled in place through OCI's stream compartment-move API before other supported updates are applied.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
