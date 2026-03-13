# API Gateway Deployment Logic Gaps

- Fixed: deployments in `CREATING`, `UPDATING`, or `DELETING` no longer report `Active`.
- Fixed: retryable deployment states now request requeue.
- Fixed: delete only completes after `GetDeployment` confirms deletion or not-found.
- Fixed: route collection equivalence is now part of the shared formal contract, so differing desired route specs require update while matching route collections skip no-op writes.

## Cluster Exercise Findings (2026-03-13)
- Delete can still wedge when OCI reports the deployment is already in `Deleted` state. During `OSOKPlatform` teardown, the controller retried `DeleteDeployment`, OCI returned `409 Conflict` with `Cannot delete deployment ... because it is in a Deleted state`, and the controller treated that as a hard delete failure instead of as delete-complete.
- The stuck `ApiGatewayDeployment` finalizer blocked further graph teardown. The CR remained present with a deletion timestamp while the parent `OSOKPlatform` stayed in `DELETING`.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
