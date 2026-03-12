# Logic Gaps

- Fixed: `CREATING` and `UPDATING` were previously reported as success instead of unsuccessful requeue.
- Fixed: bind-by-ID update paths previously relied on `status.ocid` and could target an empty ID on first reconcile.
- Fixed: delete previously completed immediately after submit instead of waiting for a follow-up `GetContainerInstance` to show deletion.
- Residual: duplicate-name garbage-collection remains best-effort and is modeled at the contract level rather than as a full OCI work-request protocol.
