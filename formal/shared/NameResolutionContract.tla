---------------------------- MODULE NameResolutionContract ----------------------------
EXTENDS TLC

BindByIDUsesSpec(capabilities, idScenario, lastMutationKind, lastMutationSource) ==
    (("bind_by_id" \in capabilities) /\ idScenario = "spec_only" /\ lastMutationKind \in {"update", "delete"}) =>
        lastMutationSource = "spec"

ResolvedNameUsesResolvedID(capabilities, idScenario, lastMutationKind, lastMutationSource) ==
    (("resolve_by_name" \in capabilities) /\ idScenario = "resolved_by_name" /\
        lastMutationKind \in {"update", "delete"}) =>
        lastMutationSource = "resolved"

=============================================================================
