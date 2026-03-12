-------------------------- MODULE CollectionEquivalenceContract --------------------------
EXTENDS TLC

CollectionDifferenceRequiresUpdate(capabilities, phase, collectionScenario, lastMutationKind) ==
    (("collection_equivalence" \in capabilities) /\ phase = "Ready" /\ collectionScenario = "different") =>
        lastMutationKind = "update"

MatchingCollectionSkipsUpdate(capabilities, phase, collectionScenario, driftScenario, lastMutationKind) ==
    (("collection_equivalence" \in capabilities) /\ phase = "Ready" /\ collectionScenario = "none" /\
        driftScenario = "none") =>
        lastMutationKind # "update"

=============================================================================
