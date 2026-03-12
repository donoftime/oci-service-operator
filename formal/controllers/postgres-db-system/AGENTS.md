# PostgresDbSystem

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
- Go implementation: `pkg/servicemanager/postgresql/`
- Property tests: `TestPropertyPostgresPendingStatesRequestRequeue`, `TestPropertyPostgresBindByIDUsesSpecIDWhenStatusIsEmpty`, `TestPropertyPostgresDeleteWaitsForConfirmedDisappearance`
- Fixed in code: `CREATING`/`UPDATING` now requeue instead of succeeding, explicit-ID updates resolve the spec ID when status is empty, and delete completion waits for confirmed disappearance.
