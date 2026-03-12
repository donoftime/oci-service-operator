# Logic Gaps

## Fixed In This Pass

- Pending `PROVISIONING` and `UPDATING` observations now request another reconcile instead of stalling.
- Delete no longer reports success immediately after submitting the OCI delete request.
- Bind-by-ID update no longer depends on a pre-populated `status.ocid`.

## Residual Risks

- The current TLA+ model is lifecycle-centric; it does not model OCI pagination or field-by-field drift.
- Rule reconciliation is modeled as atomic; OCI-side partial rule failures are not represented in the current spec.
