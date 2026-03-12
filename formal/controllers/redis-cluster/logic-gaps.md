# Logic Gaps

- Fixed: non-terminal lifecycle states no longer report `Active`.
- Fixed: bind-by-ID updates no longer depend on `status.ocid` being populated on the first reconcile.
- Fixed: delete now waits for `DELETED` or not-found before completion instead of returning done immediately.
- Fixed: secret deletion errors now block delete completion instead of being logged and ignored.
