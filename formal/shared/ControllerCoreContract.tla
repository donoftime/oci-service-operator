---------------------------- MODULE ControllerCoreContract ----------------------------
EXTENDS TLC, BaseReconcilerContract

\* Shared metadata and mutation-safety operators for richer controller specs.

ControllerFamilies ==
    {"api", "analytics", "compute", "database", "functions", "messaging", "networking", "search", "storage"}

CapabilityUniverse ==
    {"bind_by_id", "resolve_by_name", "drift_update", "confirmed_delete", "secret_write", "secret_delete",
     "paginated_resolution", "collection_equivalence", "whole_list_convergence", "best_effort_cleanup"}

ValidControllerMetadata(controllerName, family, retryableStates, activeStates, failedStates, hasSecret, capabilities) ==
    /\ controllerName \in STRING
    /\ family \in ControllerFamilies
    /\ retryableStates \subseteq STRING
    /\ activeStates \subseteq STRING
    /\ failedStates \subseteq STRING
    /\ activeStates # {}
    /\ failedStates # {}
    /\ retryableStates \cap activeStates = {}
    /\ retryableStates \cap failedStates = {}
    /\ activeStates \cap failedStates = {}
    /\ hasSecret \in BOOLEAN
    /\ capabilities \subseteq CapabilityUniverse
    /\ "confirmed_delete" \in capabilities
    /\ ("secret_delete" \in capabilities) => ("secret_write" \in capabilities)
    /\ ("paginated_resolution" \in capabilities) => ("resolve_by_name" \in capabilities)
    /\ ("whole_list_convergence" \in capabilities) => ("collection_equivalence" \in capabilities)
    /\ hasSecret = ("secret_write" \in capabilities)

KnownMutationSource(source) ==
    source \in {"status", "spec", "resolved"}

MutationUsesKnownID(lastMutationKind, lastMutationSource) ==
    (lastMutationKind \in {"update", "delete"}) => KnownMutationSource(lastMutationSource)

DeleteCompletionRequiresConfirmation(finalizerPresent, deleteConfirmed) ==
    ~finalizerPresent => deleteConfirmed

DeleteSubmittedKeepsFinalizer(deleteSubmitted, deleteConfirmed, finalizerPresent) ==
    (deleteSubmitted /\ ~deleteConfirmed) => finalizerPresent

ConfirmedDeleteRequiresResourceGone(deleteConfirmed, resourceExists) ==
    deleteConfirmed => ~resourceExists

ChooseFromOr(states, fallback) ==
    IF states # {} THEN CHOOSE state \in states: TRUE ELSE fallback

=============================================================================
