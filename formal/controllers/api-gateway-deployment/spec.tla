----------------------------- MODULE spec -----------------------------
EXTENDS TLC

INSTANCE BaseReconcilerContract

ActiveStates == {"ACTIVE"}
RetryableStates == {"CREATING", "UPDATING", "DELETING"}
FailedStates == {"FAILED", "DELETED"}

VARIABLES phase, ociState, isSuccessful, shouldRequeue, secretPresent, finalizerPresent, resourceExists

Vars == <<phase, ociState, isSuccessful, shouldRequeue, secretPresent, finalizerPresent, resourceExists>>

Init ==
    /\ phase = "Init"
    /\ ociState = "ACTIVE"
    /\ isSuccessful = TRUE
    /\ shouldRequeue = FALSE
    /\ secretPresent = FALSE
    /\ finalizerPresent = TRUE
    /\ resourceExists = TRUE

ReconcileRetryable ==
    /\ phase \in {"Init", "Ready"}
    /\ phase' = "Reconciling"
    /\ ociState' \in RetryableStates
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = TRUE
    /\ secretPresent' = FALSE
    /\ finalizerPresent' = TRUE
    /\ resourceExists' = TRUE

ReconcileActive ==
    /\ phase \in {"Init", "Reconciling"}
    /\ phase' = "Ready"
    /\ ociState' \in ActiveStates
    /\ isSuccessful' = TRUE
    /\ shouldRequeue' = FALSE
    /\ secretPresent' = FALSE
    /\ finalizerPresent' = TRUE
    /\ resourceExists' = TRUE

ReconcileFailed ==
    /\ phase \in {"Init", "Reconciling"}
    /\ phase' = "Failed"
    /\ ociState' \in FailedStates
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = FALSE
    /\ secretPresent' = FALSE
    /\ finalizerPresent' = TRUE
    /\ resourceExists' = TRUE

DeleteRequested ==
    /\ phase \in {"Ready", "Reconciling", "Failed"}
    /\ phase' = "Deleting"
    /\ ociState' = "DELETING"
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = TRUE
    /\ secretPresent' = FALSE
    /\ finalizerPresent' = TRUE
    /\ resourceExists' = TRUE

DeleteComplete ==
    /\ phase = "Deleting"
    /\ phase' = "Deleted"
    /\ ociState' = "DELETED"
    /\ isSuccessful' = FALSE
    /\ shouldRequeue' = FALSE
    /\ secretPresent' = FALSE
    /\ finalizerPresent' = FALSE
    /\ resourceExists' = FALSE

Next ==
    ReconcileRetryable \/ ReconcileActive \/ ReconcileFailed \/ DeleteRequested \/ DeleteComplete

TypeOK ==
    /\ phase \in {"Init", "Reconciling", "Ready", "Failed", "Deleting", "Deleted"}
    /\ ociState \in ActiveStates \cup RetryableStates \cup FailedStates
    /\ isSuccessful \in BOOLEAN
    /\ shouldRequeue \in BOOLEAN
    /\ secretPresent \in BOOLEAN
    /\ finalizerPresent \in BOOLEAN
    /\ resourceExists \in BOOLEAN

InvariantSuccessRequiresActive ==
    SuccessRequiresActive(ociState, isSuccessful, ActiveStates)

InvariantRetryableRequiresRequeue ==
    RetryableRequiresRequeue(ociState, shouldRequeue, RetryableStates)

InvariantDeleteRequiresResourceGone ==
    DeleteRequiresResourceGone(finalizerPresent, resourceExists)

InvariantSecretRequiresUsableState ==
    SecretRequiresUsableState(secretPresent, ociState, ActiveStates)

Spec == Init /\ [][Next]_Vars

=============================================================================
