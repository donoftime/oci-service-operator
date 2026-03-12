# Logic Gaps

- Fixed: explicit-ID binding now fails on deleted applications instead of reporting success.
- Fixed: update calls are now diff-based rather than unconditional when spec fields are merely populated.
- Fixed: delete completion now requires a follow-up `GetApplication` returning deleted/not-found.
- Accepted boundary: the model intentionally does not invent asynchronous create-progress states that the current SDK/controller surface does not expose; it reasons only over the observable `ACTIVE`, `INACTIVE`, and `DELETED` outcomes.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
