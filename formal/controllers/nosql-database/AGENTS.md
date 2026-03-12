# NoSQL Database

## Source Of Truth
- `formal/controllers/nosql-database/spec.tla`
- `formal/controllers/nosql-database/spec.cfg`

## Scope
- Controller surface: `NoSQLDatabaseReconciler`
- Service-manager surface: `pkg/servicemanager/nosql`

## Verified Properties
- Retryable table lifecycle states must request requeue.
- Successful reconcile requires an active table.
- Delete keeps the finalizer until OCI stops returning the table.
- Bind-by-ID updates must use the explicit table ID even before status is populated.

## Go Property Tests
- `TestPropertyRetryableLifecycleStatesRequeue`
- `TestPropertyBindByIDUsesExplicitSpecID`
- `TestPropertyDeleteWaitsForNotFound`

## Notes
- This controller does not create runtime secrets, so the shared secret-use invariant is vacuously satisfied.
- `logic-gaps.md` records the fixed gaps and remaining limitations.
