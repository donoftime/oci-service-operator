--------------------------- MODULE DriftAwareUpdateContract ---------------------------
EXTENDS TLC

SupportedDriftRequiresUpdate(capabilities, phase, driftScenario, lastMutationKind) ==
    (("drift_update" \in capabilities) /\ phase = "Ready" /\ driftScenario = "supported") =>
        lastMutationKind = "update"

MatchingStateSkipsUpdate(capabilities, phase, driftScenario, lastMutationKind) ==
    (("drift_update" \in capabilities) /\ phase = "Ready" /\ driftScenario = "none") =>
        lastMutationKind # "update"

=============================================================================
