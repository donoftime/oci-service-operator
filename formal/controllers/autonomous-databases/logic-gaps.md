# Logic Gaps

## Fixed
- Delete is no longer a stub; the manager now waits until OCI reports the Autonomous Database is gone before allowing finalizer removal.
- Retryable lifecycle states such as `PROVISIONING` and `UPDATING` no longer report success.
- `IsAutoScalingEnabled` and `IsFreeTier` now track explicit CR presence, so omitted booleans do not implicitly disable live settings and explicit `false` remains expressible.
- `CreatedAt` is only set when it is missing instead of using the previous inverted condition.
- Wallet secret deletion now only applies to Secrets owned by this AutonomousDatabases resource.

## Residual
- Delete progress is observed by polling `GetAutonomousDatabase` for `404`, not by OCI work-request introspection.
- Legacy wallet Secrets without ownership markers are left in place during finalization rather than being force-adopted or deleted.
