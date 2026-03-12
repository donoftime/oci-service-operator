# DataFlowApplication

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
- Go implementation: `pkg/servicemanager/dataflow/`
- Property tests: `TestPropertyDataFlowSkipsUpdateWhenSpecMatchesExistingState`, `TestPropertyDataFlowExplicitDeletedApplicationFails`, `TestPropertyDataFlowDeleteWaitsForConfirmedDisappearance`
- Fixed in code: explicit deleted applications now fail instead of binding successfully, no-op updates are skipped, and delete waits for confirmed disappearance.
