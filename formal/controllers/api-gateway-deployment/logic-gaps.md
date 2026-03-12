# API Gateway Deployment Logic Gaps

- Fixed: deployments in `CREATING`, `UPDATING`, or `DELETING` no longer report `Active`.
- Fixed: retryable deployment states now request requeue.
- Fixed: delete only completes after `GetDeployment` confirms deletion or not-found.
- Fixed: route collection equivalence is now part of the shared formal contract, so differing desired route specs require update while matching route collections skip no-op writes.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
