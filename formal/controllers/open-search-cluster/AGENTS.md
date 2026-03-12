# OpenSearchCluster

- Source of truth: `spec.tla` and `spec.cfg`
- Shared contract: `../../shared/BaseReconcilerContract.tla`
- Diagram source: `diagrams/lifecycle.puml`
- Known gaps and fix history: `logic-gaps.md`

## Verified Properties

- `TypeInvariant`
- `SuccessRequiresActiveInvariant`
- `RetryableRequiresRequeueInvariant`
- `DeleteRequiresResourceGoneInvariant`
- `SecretRequiresUsableStateInvariant`

## Notes

- This file is the controller-local knowledge log for formal verification work.
- Go implementation: `pkg/servicemanager/opensearch/`
- Property tests: `TestPropertyOpenSearchCreatePathRequestsRequeue`, `TestPropertyOpenSearchPendingStatesRequestRequeue`, `TestPropertyOpenSearchBindByIDUsesSpecIDWhenStatusIsEmpty`, `TestPropertyOpenSearchDeleteWaitsForConfirmedDisappearance`
- Fixed in code: create-by-name now requests requeue, `CREATING`/`UPDATING` no longer report success, bind/update paths preserve the target OCID, and delete waits for confirmed disappearance.
