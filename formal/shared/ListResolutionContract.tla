----------------------------- MODULE ListResolutionContract -----------------------------
EXTENDS TLC

LaterPageResolutionUsesResolvedID(capabilities, idScenario, resolutionPage, lastMutationKind, lastMutationSource) ==
    (("paginated_resolution" \in capabilities) /\ idScenario = "resolved_by_name" /\
        resolutionPage = "later_page" /\ lastMutationKind \in {"update", "delete"}) =>
        lastMutationSource = "resolved"

=============================================================================
