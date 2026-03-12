# Functions Application Logic Gaps

- Fixed: bind-by-ID updates no longer rely on a blank `status.ocid`; they use the explicit spec ID.
- Fixed: `CREATING`, `UPDATING`, and `DELETING` are treated as retryable, not successful.
- Fixed: delete only completes once a follow-up `GetApplication` shows the resource is deleted or not found.
- Residual risk: the model does not yet reason about deep config-map drift beyond the update-target contract.
