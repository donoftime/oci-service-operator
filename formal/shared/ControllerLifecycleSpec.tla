--------------------------- MODULE ControllerLifecycleSpec ---------------------------
EXTENDS TLC, BaseReconcilerContract

\* Generic controller lifecycle model used by per-controller specs.

CONSTANTS
    ControllerName,
    RetryableStates,
    ActiveStates,
    FailedStates,
    HasSecret

VARIABLES
    ociState,
    isSuccessful,
    shouldRequeue,
    finalizerPresent,
    resourceExists,
    secretPresent

vars ==
    <<ociState, isSuccessful, shouldRequeue, finalizerPresent, resourceExists, secretPresent>>

LifecycleStates ==
    RetryableStates \cup ActiveStates \cup FailedStates

ControllerMetadataInvariant ==
    /\ ControllerName \in STRING
    /\ HasSecret \in BOOLEAN
    /\ RetryableStates \subseteq STRING
    /\ ActiveStates \subseteq STRING
    /\ FailedStates \subseteq STRING

TypeInvariant ==
    /\ ControllerMetadataInvariant
    /\ ociState \in LifecycleStates
    /\ isSuccessful \in BOOLEAN
    /\ shouldRequeue \in BOOLEAN
    /\ finalizerPresent \in BOOLEAN
    /\ resourceExists \in BOOLEAN
    /\ secretPresent \in BOOLEAN

ApplyLifecycleState(state) ==
    /\ ociState' = state
    /\ resourceExists' = TRUE
    /\ finalizerPresent' = TRUE
    /\ isSuccessful' = ActiveState(state, ActiveStates)
    /\ shouldRequeue' = RetryableState(state, RetryableStates)
    /\ secretPresent' = HasSecret /\ ActiveState(state, ActiveStates)

SetLifecycleState ==
    \E state \in LifecycleStates:
        ApplyLifecycleState(state)

DeleteInProgress ==
    /\ resourceExists
    /\ \E state \in RetryableStates:
        /\ ociState' = state
        /\ resourceExists' = TRUE
        /\ finalizerPresent' = TRUE
        /\ isSuccessful' = FALSE
        /\ shouldRequeue' = TRUE
        /\ secretPresent' = FALSE

DeleteComplete ==
    /\ finalizerPresent
    /\ \E state \in FailedStates:
        /\ ociState' = state
        /\ resourceExists' = FALSE
        /\ finalizerPresent' = FALSE
        /\ isSuccessful' = FALSE
        /\ shouldRequeue' = FALSE
        /\ secretPresent' = FALSE

Init ==
    \E state \in LifecycleStates:
        /\ ociState = state
        /\ resourceExists = TRUE
        /\ finalizerPresent = TRUE
        /\ isSuccessful = ActiveState(state, ActiveStates)
        /\ shouldRequeue = RetryableState(state, RetryableStates)
        /\ secretPresent = HasSecret /\ ActiveState(state, ActiveStates)

Next ==
    \/ SetLifecycleState
    \/ DeleteInProgress
    \/ DeleteComplete

Spec ==
    Init /\ [][Next]_vars

SuccessRequiresActiveInvariant ==
    SuccessRequiresActive(ociState, isSuccessful, ActiveStates)

RetryableRequiresRequeueInvariant ==
    RetryableRequiresRequeue(ociState, shouldRequeue, RetryableStates)

DeleteRequiresResourceGoneInvariant ==
    DeleteRequiresResourceGone(finalizerPresent, resourceExists)

SecretRequiresUsableStateInvariant ==
    SecretRequiresUsableState(secretPresent, ociState, ActiveStates)

=============================================================================
