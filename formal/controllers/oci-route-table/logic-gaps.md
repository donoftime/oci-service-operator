# Logic Gaps

## Fixed In This Pass

- Pending `PROVISIONING` and `UPDATING` observations now request another reconcile instead of stalling.
- Delete no longer reports success immediately after submitting the OCI delete request.
- Bind-by-ID update no longer depends on a pre-populated `status.ocid`.
- Paginated name resolution is now part of the formal contract, so later-page matches are modeled as valid bind targets rather than being left entirely to property tests.
- The formal model now treats the desired route rule list as the supported collection surface: differing rule lists require update, matching lists skip no-op writes, and successful updates converge on full-list resubmission.

## Accepted Boundaries

- Partial OCI-side application of an already submitted full desired route-rule list remains outside the model and is handled operationally by subsequent reconciles from live state.
