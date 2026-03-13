# API Gateway Logic Gaps

- Fixed: non-terminal gateway lifecycle states (`CREATING`, `UPDATING`, `DELETING`) no longer report terminal success.
- Fixed: retryable lifecycle states now explicitly request requeue.
- Fixed: delete only completes once a follow-up `GetGateway` proves the resource is deleted or not found.
- Fixed: the formal model now covers the supported gateway update surface as diff-triggered update/no-op behavior rather than only lifecycle classification.
- Accepted boundary: gateway fields OSOK does not reconcile after creation, such as endpoint topology drift outside the supported update surface, remain outside the model.

## Cluster Exercise Findings (2026-03-13)
- Delete can still wedge when OCI reports the gateway is already in `Deleted` state. During `OSOKPlatform` teardown, the controller retried `DeleteGateway`, OCI returned `409 Conflict` with `Cannot delete gateway ... because it is in a Deleted state`, and the controller treated that as a hard delete failure instead of as delete-complete.
- The stuck `ApiGateway` finalizer blocked further graph teardown. The CR remained present with a deletion timestamp while the parent `OSOKPlatform` stayed in `DELETING`.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
