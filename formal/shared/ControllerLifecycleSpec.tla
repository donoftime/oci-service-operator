--------------------------- MODULE ControllerLifecycleSpec ---------------------------
EXTENDS TLC, BaseReconcilerContract, ControllerCoreContract, NameResolutionContract,
        DriftAwareUpdateContract, SecretSideEffectContract, ListResolutionContract,
        CollectionEquivalenceContract, WholeListConvergenceContract, BestEffortCleanupContract

\* Generic controller capability model used by per-controller specs.

CONSTANTS
    ControllerName,
    Family,
    RetryableStates,
    ActiveStates,
    FailedStates,
    HasSecret,
    Capabilities

VARIABLES
    phase,
    ociState,
    isSuccessful,
    shouldRequeue,
    finalizerPresent,
    resourceExists,
    secretPresent,
    idScenario,
    driftScenario,
    lastMutationKind,
    lastMutationSource,
    deleteSubmitted,
    deleteConfirmed,
    secretErrorMode,
    resolutionPage,
    collectionScenario,
    collectionConverged,
    cleanupScenario,
    cleanupTargetsEligible

vars ==
    <<phase, ociState, isSuccessful, shouldRequeue, finalizerPresent, resourceExists, secretPresent, idScenario,
      driftScenario, lastMutationKind, lastMutationSource, deleteSubmitted, deleteConfirmed, secretErrorMode,
      resolutionPage, collectionScenario, collectionConverged, cleanupScenario, cleanupTargetsEligible>>

LifecycleStates ==
    RetryableStates \cup ActiveStates \cup FailedStates

Supports(capability) ==
    capability \in Capabilities

AllowedIDScenarios ==
    {"status_present"}
        \cup IF Supports("bind_by_id") THEN {"spec_only"} ELSE {}
        \cup IF Supports("resolve_by_name") THEN {"resolved_by_name"} ELSE {}

AllowedDriftScenarios ==
    IF Supports("drift_update") THEN {"none", "supported"} ELSE {"none"}

AllowedResolutionPages ==
    {"unobserved"}
        \cup IF Supports("resolve_by_name") THEN {"single_page"} ELSE {}
        \cup IF Supports("paginated_resolution") THEN {"later_page"} ELSE {}

AllowedCollectionScenarios ==
    {"unobserved"}
        \cup IF Supports("collection_equivalence") \/ Supports("whole_list_convergence") THEN {"none", "different"} ELSE {}

AllowedCleanupScenarios ==
    {"unobserved"}
        \cup IF Supports("best_effort_cleanup") THEN {"none", "needed"} ELSE {}

PhaseStates ==
    {"Init", "Retryable", "Ready", "Failed", "SecretBlocked", "DeletePending", "DeleteCleanupBlocked", "Deleted"}

SourceForScenario(scenario) ==
    CASE scenario = "status_present" -> "status"
        [] scenario = "spec_only" -> "spec"
        [] scenario = "resolved_by_name" -> "resolved"
        [] OTHER -> "none"

DeletePendingState(currentState) ==
    ChooseFromOr(RetryableStates, currentState)

DeleteTerminalState(currentState) ==
    ChooseFromOr(FailedStates, currentState)

ObservedResolutionPage ==
    IF idScenario = "resolved_by_name"
        THEN ChooseFromOr(AllowedResolutionPages \ {"unobserved"}, "single_page")
        ELSE "unobserved"

ObservedCollectionScenario ==
    IF Supports("collection_equivalence") \/ Supports("whole_list_convergence")
        THEN ChooseFromOr(AllowedCollectionScenarios \ {"unobserved"}, "none")
        ELSE "unobserved"

ObservedCleanupScenario ==
    IF Supports("best_effort_cleanup")
        THEN ChooseFromOr(AllowedCleanupScenarios \ {"unobserved"}, "none")
        ELSE "unobserved"

ControllerMetadataInvariant ==
    ValidControllerMetadata(ControllerName, Family, RetryableStates, ActiveStates, FailedStates, HasSecret, Capabilities)

TypeInvariant ==
    /\ ControllerMetadataInvariant
    /\ phase \in PhaseStates
    /\ ociState \in LifecycleStates
    /\ isSuccessful \in BOOLEAN
    /\ shouldRequeue \in BOOLEAN
    /\ finalizerPresent \in BOOLEAN
    /\ resourceExists \in BOOLEAN
    /\ secretPresent \in BOOLEAN
    /\ idScenario \in AllowedIDScenarios
    /\ driftScenario \in AllowedDriftScenarios
    /\ lastMutationKind \in {"none", "update", "delete"}
    /\ lastMutationSource \in {"none", "status", "spec", "resolved"}
    /\ deleteSubmitted \in BOOLEAN
    /\ deleteConfirmed \in BOOLEAN
    /\ secretErrorMode \in {"none", "write_failed", "delete_failed", "delete_missing"}
    /\ resolutionPage \in AllowedResolutionPages
    /\ collectionScenario \in AllowedCollectionScenarios
    /\ collectionConverged \in BOOLEAN
    /\ cleanupScenario \in AllowedCleanupScenarios
    /\ cleanupTargetsEligible \in BOOLEAN

Init ==
    /\ phase = "Init"
    /\ ociState \in ActiveStates
    /\ isSuccessful = FALSE
    /\ shouldRequeue = FALSE
    /\ finalizerPresent = TRUE
    /\ resourceExists = TRUE
    /\ secretPresent = FALSE
    /\ idScenario \in AllowedIDScenarios
    /\ driftScenario \in AllowedDriftScenarios
    /\ lastMutationKind = "none"
    /\ lastMutationSource = "none"
    /\ deleteSubmitted = FALSE
    /\ deleteConfirmed = FALSE
    /\ secretErrorMode = "none"
    /\ resolutionPage = "unobserved"
    /\ collectionScenario = "unobserved"
    /\ collectionConverged = FALSE
    /\ cleanupScenario = "unobserved"
    /\ cleanupTargetsEligible = TRUE

ObserveRetryable ==
    /\ phase = "Init"
    /\ RetryableStates # {}
    /\ \E state \in RetryableStates:
        /\ phase' = "Retryable"
        /\ ociState' = state
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = TRUE
    /\ finalizerPresent' = TRUE
    /\ resourceExists' = TRUE
    /\ secretPresent' = FALSE
    /\ UNCHANGED <<idScenario, driftScenario>>
    /\ lastMutationKind' = "none"
    /\ lastMutationSource' = "none"
    /\ deleteSubmitted' = FALSE
    /\ deleteConfirmed' = FALSE
    /\ secretErrorMode' = "none"
    /\ resolutionPage' = ObservedResolutionPage
    /\ collectionScenario' = "unobserved"
    /\ collectionConverged' = FALSE
    /\ cleanupScenario' = "unobserved"
    /\ cleanupTargetsEligible' = TRUE

ObserveActiveReady ==
    /\ phase = "Init"
    /\ LET state == CHOOSE activeState \in ActiveStates: TRUE
           collectionScenarioValue == ObservedCollectionScenario
           cleanupScenarioValue == ObservedCleanupScenario
           mutationKind ==
               IF (Supports("drift_update") /\ driftScenario = "supported") \/ (collectionScenarioValue = "different")
                   THEN "update"
                   ELSE "none"
           mutationSource == IF mutationKind = "update" THEN SourceForScenario(idScenario) ELSE "none"
       IN /\ lastMutationKind' = mutationKind
          /\ lastMutationSource' = mutationSource
          /\ phase' = "Ready"
          /\ ociState' = state
          /\ collectionScenario' = collectionScenarioValue
          /\ collectionConverged' =
              IF collectionScenarioValue = "different"
                  THEN mutationKind = "update"
                  ELSE TRUE
          /\ cleanupScenario' = cleanupScenarioValue
          /\ cleanupTargetsEligible' = TRUE
    /\ isSuccessful' = TRUE
    /\ shouldRequeue' = FALSE
    /\ finalizerPresent' = TRUE
    /\ resourceExists' = TRUE
    /\ secretPresent' = HasSecret
    /\ UNCHANGED <<idScenario, driftScenario>>
    /\ deleteSubmitted' = FALSE
    /\ deleteConfirmed' = FALSE
    /\ secretErrorMode' = "none"
    /\ resolutionPage' = ObservedResolutionPage

ObserveActiveSecretWriteFailure ==
    /\ phase = "Init"
    /\ Supports("secret_write")
    /\ LET state == CHOOSE activeState \in ActiveStates: TRUE
           collectionScenarioValue == ObservedCollectionScenario
           mutationKind ==
               IF (Supports("drift_update") /\ driftScenario = "supported") \/ (collectionScenarioValue = "different")
                   THEN "update"
                   ELSE "none"
           mutationSource == IF mutationKind = "update" THEN SourceForScenario(idScenario) ELSE "none"
       IN /\ lastMutationKind' = mutationKind
          /\ lastMutationSource' = mutationSource
          /\ phase' = "SecretBlocked"
          /\ ociState' = state
          /\ collectionScenario' = collectionScenarioValue
          /\ collectionConverged' =
              IF collectionScenarioValue = "different"
                  THEN mutationKind = "update"
                  ELSE TRUE
          /\ cleanupScenario' = "unobserved"
          /\ cleanupTargetsEligible' = TRUE
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = TRUE
    /\ finalizerPresent' = TRUE
    /\ resourceExists' = TRUE
    /\ secretPresent' = FALSE
    /\ UNCHANGED <<idScenario, driftScenario>>
    /\ deleteSubmitted' = FALSE
    /\ deleteConfirmed' = FALSE
    /\ secretErrorMode' = "write_failed"
    /\ resolutionPage' = ObservedResolutionPage

ObserveFailed ==
    /\ phase = "Init"
    /\ \E state \in FailedStates:
        /\ phase' = "Failed"
        /\ ociState' = state
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = FALSE
    /\ finalizerPresent' = TRUE
    /\ resourceExists' = TRUE
    /\ secretPresent' = FALSE
    /\ UNCHANGED <<idScenario, driftScenario>>
    /\ lastMutationKind' = "none"
    /\ lastMutationSource' = "none"
    /\ deleteSubmitted' = FALSE
    /\ deleteConfirmed' = FALSE
    /\ secretErrorMode' = "none"
    /\ resolutionPage' = ObservedResolutionPage
    /\ collectionScenario' = "unobserved"
    /\ collectionConverged' = FALSE
    /\ cleanupScenario' = "unobserved"
    /\ cleanupTargetsEligible' = TRUE

RequestDelete ==
    /\ phase \in {"Ready", "Retryable", "Failed", "SecretBlocked"}
    /\ resourceExists
    /\ phase' = "DeletePending"
    /\ ociState' = DeletePendingState(ociState)
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = TRUE
    /\ finalizerPresent' = TRUE
    /\ resourceExists' = TRUE
    /\ secretPresent' = FALSE
    /\ lastMutationKind' = "delete"
    /\ lastMutationSource' = SourceForScenario(idScenario)
    /\ UNCHANGED <<idScenario, driftScenario, resolutionPage, collectionScenario, collectionConverged,
        cleanupScenario, cleanupTargetsEligible>>
    /\ deleteSubmitted' = TRUE
    /\ deleteConfirmed' = FALSE
    /\ secretErrorMode' = "none"

ConfirmDelete ==
    /\ phase = "DeletePending"
    /\ phase' = "Deleted"
    /\ ociState' = DeleteTerminalState(ociState)
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = FALSE
    /\ finalizerPresent' = FALSE
    /\ resourceExists' = FALSE
    /\ secretPresent' = FALSE
    /\ lastMutationKind' = "delete"
    /\ lastMutationSource' = SourceForScenario(idScenario)
    /\ UNCHANGED <<idScenario, driftScenario, resolutionPage, collectionScenario, collectionConverged,
        cleanupScenario, cleanupTargetsEligible>>
    /\ deleteSubmitted' = TRUE
    /\ deleteConfirmed' = TRUE
    /\ secretErrorMode' = "none"

ConfirmDeleteMissingSecret ==
    /\ phase = "DeletePending"
    /\ Supports("secret_delete")
    /\ phase' = "Deleted"
    /\ ociState' = DeleteTerminalState(ociState)
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = FALSE
    /\ finalizerPresent' = FALSE
    /\ resourceExists' = FALSE
    /\ secretPresent' = FALSE
    /\ lastMutationKind' = "delete"
    /\ lastMutationSource' = SourceForScenario(idScenario)
    /\ UNCHANGED <<idScenario, driftScenario, resolutionPage, collectionScenario, collectionConverged,
        cleanupScenario, cleanupTargetsEligible>>
    /\ deleteSubmitted' = TRUE
    /\ deleteConfirmed' = TRUE
    /\ secretErrorMode' = "delete_missing"

ConfirmDeleteSecretDeleteFailure ==
    /\ phase = "DeletePending"
    /\ Supports("secret_delete")
    /\ phase' = "DeleteCleanupBlocked"
    /\ ociState' = DeleteTerminalState(ociState)
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = TRUE
    /\ finalizerPresent' = TRUE
    /\ resourceExists' = FALSE
    /\ secretPresent' = FALSE
    /\ lastMutationKind' = "delete"
    /\ lastMutationSource' = SourceForScenario(idScenario)
    /\ UNCHANGED <<idScenario, driftScenario, resolutionPage, collectionScenario, collectionConverged,
        cleanupScenario, cleanupTargetsEligible>>
    /\ deleteSubmitted' = TRUE
    /\ deleteConfirmed' = TRUE
    /\ secretErrorMode' = "delete_failed"

RecoverSecretCleanup ==
    /\ phase = "DeleteCleanupBlocked"
    /\ phase' = "Deleted"
    /\ ociState' = DeleteTerminalState(ociState)
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = FALSE
    /\ finalizerPresent' = FALSE
    /\ resourceExists' = FALSE
    /\ secretPresent' = FALSE
    /\ UNCHANGED <<idScenario, driftScenario, lastMutationKind, lastMutationSource, deleteSubmitted, deleteConfirmed,
        resolutionPage, collectionScenario, collectionConverged, cleanupScenario, cleanupTargetsEligible>>
    /\ secretErrorMode' = "none"

StayRetryable ==
    /\ phase = "Retryable"
    /\ UNCHANGED vars

StayReady ==
    /\ phase = "Ready"
    /\ UNCHANGED vars

StayFailed ==
    /\ phase = "Failed"
    /\ UNCHANGED vars

StaySecretBlocked ==
    /\ phase = "SecretBlocked"
    /\ UNCHANGED vars

StayDeleteCleanupBlocked ==
    /\ phase = "DeleteCleanupBlocked"
    /\ UNCHANGED vars

StayDeleted ==
    /\ phase = "Deleted"
    /\ UNCHANGED vars

Next ==
    \/ ObserveRetryable
    \/ ObserveActiveReady
    \/ ObserveActiveSecretWriteFailure
    \/ ObserveFailed
    \/ RequestDelete
    \/ ConfirmDelete
    \/ ConfirmDeleteMissingSecret
    \/ ConfirmDeleteSecretDeleteFailure
    \/ RecoverSecretCleanup
    \/ StayRetryable
    \/ StayReady
    \/ StayFailed
    \/ StaySecretBlocked
    \/ StayDeleteCleanupBlocked
    \/ StayDeleted

Spec ==
    Init /\ [][Next]_vars

SuccessRequiresActiveInvariant ==
    SuccessRequiresActive(ociState, isSuccessful, ActiveStates)

RetryableRequiresRequeueInvariant ==
    RetryableRequiresRequeue(ociState, shouldRequeue, RetryableStates)

DeleteRequiresResourceGoneInvariant ==
    DeleteRequiresResourceGone(finalizerPresent, resourceExists)

MutationUsesBoundIDInvariant ==
    MutationUsesKnownID(lastMutationKind, lastMutationSource)

DeleteRequiresConfirmationInvariant ==
    DeleteCompletionRequiresConfirmation(finalizerPresent, deleteConfirmed)

DeleteSubmittedKeepsFinalizerInvariant ==
    DeleteSubmittedKeepsFinalizer(deleteSubmitted, deleteConfirmed, finalizerPresent)

ConfirmedDeleteRemovesResourceInvariant ==
    ConfirmedDeleteRequiresResourceGone(deleteConfirmed, resourceExists)

BindByIDUsesSpecInvariant ==
    BindByIDUsesSpec(Capabilities, idScenario, lastMutationKind, lastMutationSource)

ResolvedNameUsesResolvedIDInvariant ==
    ResolvedNameUsesResolvedID(Capabilities, idScenario, lastMutationKind, lastMutationSource)

LaterPageResolutionUsesResolvedIDInvariant ==
    LaterPageResolutionUsesResolvedID(Capabilities, idScenario, resolutionPage, lastMutationKind, lastMutationSource)

SupportedDriftRequiresUpdateInvariant ==
    SupportedDriftRequiresUpdate(Capabilities, phase, driftScenario, lastMutationKind)

MatchingStateSkipsUpdateInvariant ==
    MatchingStateSkipsUpdate(Capabilities, phase, driftScenario, lastMutationKind)

CollectionDifferenceRequiresUpdateInvariant ==
    CollectionDifferenceRequiresUpdate(Capabilities, phase, collectionScenario, lastMutationKind)

MatchingCollectionSkipsUpdateInvariant ==
    MatchingCollectionSkipsUpdate(Capabilities, phase, collectionScenario, driftScenario, lastMutationKind)

WholeListConvergesAfterUpdateInvariant ==
    WholeListConvergesAfterUpdate(Capabilities, phase, collectionScenario, lastMutationKind, collectionConverged)

SecretRequiresUsableStateInvariant ==
    SecretRequiresUsableState(secretPresent, ociState, ActiveStates)

SecretWriteFailuresBlockSuccessInvariant ==
    SecretWriteFailuresBlockSuccess(Capabilities, phase, secretErrorMode, isSuccessful)

SecretDeleteFailuresBlockCompletionInvariant ==
    SecretDeleteFailuresBlockCompletion(Capabilities, phase, secretErrorMode, finalizerPresent, deleteConfirmed,
        resourceExists)

MissingSecretAllowsDeleteInvariant ==
    MissingSecretAllowsDelete(Capabilities, phase, secretErrorMode, finalizerPresent)

BestEffortCleanupKeepsSuccessInvariant ==
    BestEffortCleanupKeepsSuccess(Capabilities, phase, cleanupScenario, isSuccessful)

CleanupTargetsStayEligibleInvariant ==
    CleanupTargetsStayEligible(Capabilities, cleanupScenario, cleanupTargetsEligible)

=============================================================================
