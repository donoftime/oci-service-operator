------------------------------- MODULE spec -------------------------------
EXTENDS ControllerLifecycleSpec

StatusPresentUsesStatusInvariant ==
    (idScenario = "status_present" /\ lastMutationKind \in {"update", "delete"}) =>
        lastMutationSource = "status"

=============================================================================
