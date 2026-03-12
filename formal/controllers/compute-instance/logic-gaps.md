# Logic Gaps

- Fixed: `PROVISIONING`, `STARTING`, and `STOPPING` no longer report terminal success.
- Fixed: bind-by-ID update paths now target the explicit spec OCID even when `status.ocid` is unset.
- Fixed: delete no longer completes immediately after terminate submission; it waits for `TERMINATED` or not-found.
- Fixed: bound-instance reconciles now diff and apply freeform/defined tag drift instead of limiting updates to display name only.
- Accepted boundary: the formal model intentionally stops at the fields OSOK actually reconciles on bound instances: display name plus freeform/defined tags. Shape, image, subnet, and instance-option drift remain outside the model because the controller does not update those fields after bind.
