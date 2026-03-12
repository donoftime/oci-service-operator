# MySQL DB System

## Source Of Truth
- `formal/controllers/mysql-dbsystem/spec.tla`
- `formal/controllers/mysql-dbsystem/spec.cfg`

## Scope
- Controller surface: `MySqlDBsystemReconciler`
- Service-manager surface: `pkg/servicemanager/mysql/dbsystem`

## Verified Properties
- Retryable OCI lifecycle states must requeue instead of reporting success.
- Secret creation is only allowed once the DB system is usable.
- Finalizers are only removed after OCI no longer returns the DB system.
- Failed lifecycles must not produce successful reconcile results.

## Go Property Tests
- `TestPropertyRetryableLifecycleStatesRequeue`
- `TestPropertyDeleteWaitsForResourceToDisappear`

## Notes
- The formal model captures the controller contract around lifecycle classification, secret creation, and delete completion.
- `logic-gaps.md` records the implementation changes behind those properties.
