--------------------------- MODULE BaseReconcilerContract ---------------------------
EXTENDS TLC

\* Shared helper operators for per-controller specs.

RetryableState(ociState, retryableStates) ==
    ociState \in retryableStates

ActiveState(ociState, activeStates) ==
    ociState \in activeStates

FailedState(ociState, failedStates) ==
    ociState \in failedStates

SuccessRequiresActive(ociState, isSuccessful, activeStates) ==
    isSuccessful => ActiveState(ociState, activeStates)

RetryableRequiresRequeue(ociState, shouldRequeue, retryableStates) ==
    RetryableState(ociState, retryableStates) => shouldRequeue

DeleteRequiresResourceGone(finalizerPresent, resourceExists) ==
    finalizerPresent \/ ~resourceExists

SecretRequiresUsableState(secretPresent, ociState, activeStates) ==
    secretPresent => ActiveState(ociState, activeStates)

=============================================================================
