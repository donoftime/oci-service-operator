# OCI Queue Logic Gaps

- Fixed: async creation and non-terminal queue states now explicitly request requeue.
- Fixed: bind-by-ID updates no longer depend on a pre-populated `status.ocid`.
- Fixed: delete only completes once `GetQueue` confirms deletion or not-found.
- Fixed: supported queue update reconciliation now includes tag drift in addition to the existing display name and timeout/visibility settings.
- Accepted boundary: the formal model covers the queue fields OSOK reconciles after create (`DisplayName`, visibility, timeout, dead-letter delivery count, and freeform/defined tags). Retention and encryption-key drift remain outside the model because the controller does not update those fields.
