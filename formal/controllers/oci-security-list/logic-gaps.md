# Logic Gaps

## Fixed In This Pass

- Pending `PROVISIONING` and `UPDATING` observations now request another reconcile instead of stalling.
- Delete no longer reports success immediately after submitting the OCI delete request.
- Bind-by-ID update no longer depends on a pre-populated `status.ocid`.
- Paginated name resolution is now part of the formal contract, so later-page matches are modeled as valid bind targets rather than being left entirely to property tests.
- The formal model now treats the desired ingress and egress rule lists as the supported collection surface: differing rule sets require update, matching rule sets skip no-op writes, and successful updates converge on full-list resubmission.
- Managed security-list reconciles now continue using `status.ocid` after create/bind, so supported drift updates do not fall back to create-by-name behavior.
- `spec.compartmentId` drift is now reconciled in place through OCI's security-list compartment-move API before rule/tag reconciliation is applied.

## Accepted Boundaries

- Partial OCI-side application of an already submitted full desired ingress/egress rule set remains outside the model and is handled operationally by subsequent reconciles from live state.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
