# Functions Function Logic Gaps

- Fixed: bind-by-ID updates no longer rely on a blank `status.ocid`; they use the explicit spec ID.
- Fixed: retryable lifecycle states no longer create endpoint secrets or report success.
- Fixed: secret creation failures and secret deletion failures now block reconcile completion instead of being swallowed.
- Fixed: delete only completes once `GetFunction` confirms deletion or not-found.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
