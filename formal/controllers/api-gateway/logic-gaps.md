# API Gateway Logic Gaps

- Fixed: non-terminal gateway lifecycle states (`CREATING`, `UPDATING`, `DELETING`) no longer report terminal success.
- Fixed: retryable lifecycle states now explicitly request requeue.
- Fixed: delete only completes once a follow-up `GetGateway` proves the resource is deleted or not found.
- Fixed: the formal model now covers the supported gateway update surface as diff-triggered update/no-op behavior rather than only lifecycle classification.
- Accepted boundary: gateway fields OSOK does not reconcile after creation, such as endpoint topology drift outside the supported update surface, remain outside the model.
