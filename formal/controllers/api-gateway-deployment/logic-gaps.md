# API Gateway Deployment Logic Gaps

- Fixed: deployments in `CREATING`, `UPDATING`, or `DELETING` no longer report `Active`.
- Fixed: retryable deployment states now request requeue.
- Fixed: delete only completes after `GetDeployment` confirms deletion or not-found.
- Residual risk: the model focuses on lifecycle/finalizer correctness, not full route-spec equivalence.
