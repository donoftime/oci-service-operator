--------------------------- MODULE BestEffortCleanupContract ---------------------------
EXTENDS TLC

BestEffortCleanupKeepsSuccess(capabilities, phase, cleanupScenario, isSuccessful) ==
    (("best_effort_cleanup" \in capabilities) /\ phase = "Ready" /\ cleanupScenario = "needed") =>
        isSuccessful

CleanupTargetsStayEligible(capabilities, cleanupScenario, cleanupTargetsEligible) ==
    (("best_effort_cleanup" \in capabilities) /\ cleanupScenario = "needed") =>
        cleanupTargetsEligible

=============================================================================
