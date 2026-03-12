# ContainerInstance

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
- Go implementation: `pkg/servicemanager/containerinstance/`
- Property tests: `TestPropertyContainerInstancePendingStatesRequestRequeue`, `TestPropertyContainerInstanceBindByIDUsesSpecIDWhenStatusIsEmpty`, `TestPropertyContainerInstanceDeleteWaitsForConfirmedDisappearance`
- Fixed in code: non-terminal lifecycle states now requeue, bind-by-ID updates resolve the spec ID when status is blank, and delete waits for confirmed disappearance before completion.
