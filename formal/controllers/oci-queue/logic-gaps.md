# OCI Queue Logic Gaps

- Fixed: async creation and non-terminal queue states now explicitly request requeue.
- Fixed: bind-by-ID updates no longer depend on a pre-populated `status.ocid`.
- Fixed: delete only completes once `GetQueue` confirms deletion or not-found.
- Residual risk: the model does not yet cover every queue attribute drift dimension, only lifecycle/finalizer correctness.
