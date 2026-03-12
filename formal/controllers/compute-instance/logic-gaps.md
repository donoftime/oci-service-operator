# Logic Gaps

- Fixed: `PROVISIONING`, `STARTING`, and `STOPPING` no longer report terminal success.
- Fixed: bind-by-ID update paths now target the explicit spec OCID even when `status.ocid` is unset.
- Fixed: delete no longer completes immediately after terminate submission; it waits for `TERMINATED` or not-found.
- Residual: the model is contract-focused and does not include the full instance-option or shape-drift surface.
