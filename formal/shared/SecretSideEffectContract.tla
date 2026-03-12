--------------------------- MODULE SecretSideEffectContract ---------------------------
EXTENDS TLC

SecretWriteFailuresBlockSuccess(capabilities, phase, secretErrorMode, isSuccessful) ==
    (("secret_write" \in capabilities) /\ secretErrorMode = "write_failed") =>
        /\ phase = "SecretBlocked"
        /\ ~isSuccessful

SecretDeleteFailuresBlockCompletion(capabilities, phase, secretErrorMode, finalizerPresent, deleteConfirmed,
resourceExists) ==
    (("secret_delete" \in capabilities) /\ secretErrorMode = "delete_failed") =>
        /\ phase = "DeleteCleanupBlocked"
        /\ finalizerPresent
        /\ deleteConfirmed
        /\ ~resourceExists

MissingSecretAllowsDelete(capabilities, phase, secretErrorMode, finalizerPresent) ==
    (("secret_delete" \in capabilities) /\ secretErrorMode = "delete_missing") =>
        /\ phase = "Deleted"
        /\ ~finalizerPresent

=============================================================================
