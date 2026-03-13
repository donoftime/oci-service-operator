# Stream Logic Gaps

- Fixed: existing-by-name updates now target the resolved stream ID instead of a blank spec ID.
- Fixed: `FAILED` and `DELETED` streams no longer report terminal success.
- Fixed: `CREATING`, `UPDATING`, and `DELETING` now requeue instead of silently stalling.
- Fixed: delete resolves IDs from spec, status, or name lookup and only completes once the stream is gone.
- Fixed: secret generation now fails safely when `MessagesEndpoint` is missing instead of panicking.
- Fixed: managed stream reconciles now continue using `status.ocid` after create/name-resolution, so supported drift updates do not fall back to fresh name lookup.
- Fixed: `spec.compartmentId` drift is now reconciled in place through OCI's stream compartment-move API before other supported updates are applied.

## Cluster Exercise Findings (2026-03-13)
- Successful update reconciles can leave the most recent CR condition at `Updating`. During the `no_reap=true` tag exercise, the controller logged `Stream my-platform-stream is updated successfully`, later reconciled the stream as active, and recreated the connection Secret, but `status.status.conditions[-1]` still remained `Updating` with message `Stream Update success`.
- Delete can also wedge after the provider reaches a terminal state. During `OSOKPlatform` teardown, OCI returned `400 InvalidParameter` with `Stream ... is DELETED`, but the controller treated that as a hard delete failure instead of as delete-complete, so the CR remained stuck on its finalizer.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
