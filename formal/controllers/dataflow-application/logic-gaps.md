# Logic Gaps

- Fixed: explicit-ID binding now fails on deleted applications instead of reporting success.
- Fixed: update calls are now diff-based rather than unconditional when spec fields are merely populated.
- Fixed: delete completion now requires a follow-up `GetApplication` returning deleted/not-found.
- Residual: OCI Data Flow applications expose only `ACTIVE`, `INACTIVE`, and `DELETED` in this SDK surface, so the model tracks contract safety rather than async create work requests.
