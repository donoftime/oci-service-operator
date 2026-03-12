# Logic Gaps

## Fixed
- `FAILED` lifecycle now returns an unsuccessful reconcile result instead of looking successful.
- Retryable states such as `CREATING`, `UPDATING`, and `INACTIVE` now request requeue rather than silently succeeding.
- Delete is no longer a no-op; it waits for the DB system to disappear before allowing finalizer removal.
- Secret cleanup now happens after OCI confirms the DB system is gone.

## Residual
- Delete completion is detected through `GetDbSystem` returning `404`, not through work-request inspection.
- Non-lifecycle OCI read failures still propagate as errors and rely on controller retry policy rather than custom throttling logic.
