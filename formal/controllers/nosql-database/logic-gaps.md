# Logic Gaps

## Fixed
- Non-terminal table states such as `CREATING` and `UPDATING` now requeue instead of reporting success.
- Bind-by-ID updates now target the explicit `spec.tableId` when `status.ocid` is still empty.
- Delete now waits until `GetTable` reports `404` before allowing finalizer removal.

## Residual
- Delete completion is detected through repeated `GetTable` reads rather than work-request tracking.
- Table updates still treat any provided DDL or limits as desired state and do not diff against every OCI field.
