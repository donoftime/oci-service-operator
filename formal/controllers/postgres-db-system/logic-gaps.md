# Logic Gaps

- Fixed: non-terminal lifecycle states no longer report `Active`.
- Fixed: bind-by-ID updates no longer depend on a pre-populated `status.ocid`.
- Fixed: delete now waits for not-found before completion instead of clearing immediately after submit.
- Fixed: secret deletion errors now block delete completion instead of being logged and ignored.
