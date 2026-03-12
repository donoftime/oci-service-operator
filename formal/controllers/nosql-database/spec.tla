------------------------------ MODULE spec ------------------------------
EXTENDS TLC, BaseReconcilerContract

CONSTANTS RetryableStates, ActiveStates, FailedStates

LifecycleStates == RetryableStates \cup ActiveStates \cup FailedStates

VARIABLES ociState, isSuccessful, shouldRequeue, finalizerPresent, resourceExists, secretPresent

vars == <<ociState, isSuccessful, shouldRequeue, finalizerPresent, resourceExists, secretPresent>>

Init ==
    /\ ociState \in LifecycleStates
    /\ isSuccessful = ActiveState(ociState, ActiveStates)
    /\ shouldRequeue = RetryableState(ociState, RetryableStates)
    /\ finalizerPresent = TRUE
    /\ resourceExists = TRUE
    /\ secretPresent = FALSE

ClassifyLifecycle ==
    /\ ociState' = ociState
    /\ finalizerPresent' = finalizerPresent
    /\ resourceExists' = resourceExists
    /\ IF RetryableState(ociState, RetryableStates)
          THEN /\ isSuccessful' = FALSE
               /\ shouldRequeue' = TRUE
               /\ secretPresent' = FALSE
       ELSE IF ActiveState(ociState, ActiveStates)
          THEN /\ isSuccessful' = TRUE
               /\ shouldRequeue' = FALSE
               /\ secretPresent' = FALSE
          ELSE /\ isSuccessful' = FALSE
               /\ shouldRequeue' = FALSE
               /\ secretPresent' = FALSE

RequestDelete ==
    /\ resourceExists
    /\ resourceExists' \in BOOLEAN
    /\ finalizerPresent' = resourceExists'
    /\ UNCHANGED <<ociState, isSuccessful, shouldRequeue, secretPresent>>

Next == ClassifyLifecycle \/ RequestDelete

SuccessRequiresActiveInv == SuccessRequiresActive(ociState, isSuccessful, ActiveStates)
RetryableRequiresRequeueInv == RetryableRequiresRequeue(ociState, shouldRequeue, RetryableStates)
DeleteRequiresResourceGoneInv == DeleteRequiresResourceGone(finalizerPresent, resourceExists)
SecretRequiresUsableStateInv == SecretRequiresUsableState(secretPresent, ociState, ActiveStates)

Spec == Init /\ [][Next]_vars

=============================================================================
