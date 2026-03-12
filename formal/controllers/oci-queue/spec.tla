------------------------------- MODULE spec -------------------------------
EXTENDS ControllerLifecycleSpec

\* Queue-specific drift is modeled through the shared drift_update contract.
\* Supported in-place updates include display name, timeout/visibility, dead-letter count,
\* tags, and non-empty custom encryption key changes. retentionInSeconds stays outside
\* the drift model because the controller rejects it before any OCI mutation.

StatusPresentUsesStatusInvariant ==
    (idScenario = "status_present" /\ lastMutationKind \in {"update", "delete"}) =>
        lastMutationSource = "status"

=============================================================================
