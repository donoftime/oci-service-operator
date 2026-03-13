# OciQueue

- Source of truth: `spec.tla` and `spec.cfg`
- Shared contracts: `../../shared/ControllerCoreContract.tla`, `../../shared/NameResolutionContract.tla`,
  `../../shared/ListResolutionContract.tla`, `../../shared/DriftAwareUpdateContract.tla`,
  `../../shared/CollectionEquivalenceContract.tla`, `../../shared/WholeListConvergenceContract.tla`,
  `../../shared/BestEffortCleanupContract.tla`, `../../shared/SecretSideEffectContract.tla`
- Diagram sources: `diagrams/activity.puml`, `diagrams/sequence.puml`, `diagrams/state-machine.puml`
- Known gaps and fix history: `logic-gaps.md`
- Capabilities: `bind_by_id,resolve_by_name,drift_update,confirmed_delete,secret_write,secret_delete`

## Verified Properties

- `ControllerMetadataInvariant`
- `TypeInvariant`
- `SuccessRequiresActiveInvariant`
- `RetryableRequiresRequeueInvariant`
- `DeleteRequiresResourceGoneInvariant`
- `MutationUsesBoundIDInvariant`
- `StatusPresentUsesStatusInvariant`
- `DeleteRequiresConfirmationInvariant`
- `DeleteSubmittedKeepsFinalizerInvariant`
- `ConfirmedDeleteRemovesResourceInvariant`
- `BindByIDUsesSpecInvariant`
- `ResolvedNameUsesResolvedIDInvariant`
- `LaterPageResolutionUsesResolvedIDInvariant`
- `SupportedDriftRequiresUpdateInvariant`
- `MatchingStateSkipsUpdateInvariant`
- `CollectionDifferenceRequiresUpdateInvariant`
- `MatchingCollectionSkipsUpdateInvariant`
- `WholeListConvergesAfterUpdateInvariant`
- `SecretRequiresUsableStateInvariant`
- `SecretWriteFailuresBlockSuccessInvariant`
- `SecretDeleteFailuresBlockCompletionInvariant`
- `MissingSecretAllowsDeleteInvariant`
- `BestEffortCleanupKeepsSuccessInvariant`
- `CleanupTargetsStayEligibleInvariant`

## Notes

- This file is the controller-local knowledge log for formal verification work.
- Update it with controller-specific counterexamples, linked Go property tests, and the final code fixes.
- Queue-specific property coverage for tracked-ID updates, compartment moves, custom-encryption-key drift, and retention rejection lives in `pkg/servicemanager/queue/queue_properties_test.go`.
