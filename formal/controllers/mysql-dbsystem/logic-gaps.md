# Logic Gaps

## Fixed
- `FAILED` lifecycle now returns an unsuccessful reconcile result instead of looking successful.
- Retryable states such as `CREATING`, `UPDATING`, and `INACTIVE` now request requeue rather than silently succeeding.
- Delete is no longer a no-op; it now tracks matching OCI delete work requests and only completes after the work request succeeds or OCI already reports the DB system missing.
- Secret cleanup now happens after OCI delete completion is confirmed by work-request success or by OCI reporting the DB system missing.
- Transient non-lifecycle OCI read failures such as throttling and 5xx responses now request controller-managed requeue instead of surfacing as hard reconcile errors.
