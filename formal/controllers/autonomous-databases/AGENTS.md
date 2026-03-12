# Autonomous Databases

## Source Of Truth
- `formal/controllers/autonomous-databases/spec.tla`
- `formal/controllers/autonomous-databases/spec.cfg`

## Scope
- Controller surface: `AutonomousDatabasesReconciler`
- Service-manager surface: `pkg/servicemanager/autonomousdatabases/adb`

## Verified Properties
- Retryable OCI lifecycle states must return `ShouldRequeue`.
- Terminal success is only allowed for usable OCI states.
- Finalizers are only cleared once the OCI resource is gone.
- Wallet secrets are only materialized for usable OCI states.

## Go Property Tests
- `TestPropertyRetryableLifecycleStatesRequeue`
- `TestPropertyExplicitFalseBooleansTriggerUpdate`
- `TestPropertyOmittedFalseBooleansDoNotTriggerUpdate`
- `TestPropertySpecJSONTracksExplicitADBBooleans`
- `TestPropertyDeleteWaitsForResourceToDisappear`

## Notes
- The TLA+ model captures the controller contract rather than every OCI field.
- `logic-gaps.md` records the concrete bugs fixed in the Go implementation and the remaining caveats.
