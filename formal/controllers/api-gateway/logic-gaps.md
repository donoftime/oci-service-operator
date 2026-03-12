# API Gateway Logic Gaps

- Fixed: non-terminal gateway lifecycle states (`CREATING`, `UPDATING`, `DELETING`) no longer report terminal success.
- Fixed: retryable lifecycle states now explicitly request requeue.
- Fixed: delete only completes once a follow-up `GetGateway` proves the resource is deleted or not found.
- Residual risk: the spec models lifecycle classification and finalizer safety, not full OCI request-shape drift coverage.
