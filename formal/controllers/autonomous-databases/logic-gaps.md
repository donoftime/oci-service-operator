# Logic Gaps

## Fixed
- Delete is no longer a stub; the manager now waits until OCI reports the Autonomous Database is gone before allowing finalizer removal.
- Retryable lifecycle states such as `PROVISIONING` and `UPDATING` no longer report success.
- `IsAutoScalingEnabled` and `IsFreeTier` now track explicit CR presence, so omitted booleans do not implicitly disable live settings and explicit `false` remains expressible.
- `CreatedAt` is only set when it is missing instead of using the previous inverted condition.
- Wallet secret deletion now only applies to Secrets owned by this AutonomousDatabases resource.

## Fixed In This Pass
- Delete submission now captures and logs the returned OCI work-request ID for operator visibility while the controller waits for OCI to remove the database.
- Legacy wallet Secrets without OSOK ownership markers are now explicitly inspected and logged during finalization so their preserve-by-default behavior is visible instead of implicit.
- Managed and bound ADB reconciles now persist and reuse the tracked OCID before update, so supported update drift is applied to resolved resources instead of only create-time paths.
- `spec.compartmentId` drift is now reconciled in place through OCI's Autonomous Database compartment-move API before supported updates are applied.
- `spec.computeModel` and `spec.computeCount` drift now flow through `UpdateAutonomousDatabaseDetails` instead of remaining create-only fields.
- `spec.adminPassword` now flows through `UpdateAutonomousDatabaseDetails` when an admin-password Secret is configured, which keeps password rotation on the supported update surface despite OCI not returning a diffable live password value.
- `spec.dbName` drift is now rejected before mutation instead of being sent through `UpdateAutonomousDatabaseDetails`.

## Accepted Boundaries
- The vendored OCI Database SDK exposes delete work-request IDs but does not provide generated work-request lookup APIs, so delete completion still relies on follow-up `GetAutonomousDatabase` calls until OCI returns `404`.
- Legacy wallet Secrets without OSOK ownership markers remain preserved by design; this controller does not force-adopt or delete pre-existing user-managed Secrets.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
