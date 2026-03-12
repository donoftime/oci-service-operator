-------------------------- MODULE WholeListConvergenceContract --------------------------
EXTENDS TLC

WholeListConvergesAfterUpdate(capabilities, phase, collectionScenario, lastMutationKind, collectionConverged) ==
    (("whole_list_convergence" \in capabilities) /\ phase = "Ready" /\ collectionScenario = "different") =>
        /\ lastMutationKind = "update"
        /\ collectionConverged

=============================================================================
