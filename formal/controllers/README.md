# Controller Specs

Each subdirectory corresponds to one registered controller slug and contains:

- `spec.tla`
- `spec.cfg`
- `logic-gaps.md`
- `AGENTS.md`
- `diagrams/*.puml`

The top-level verification commands walk this tree directly.

Each controller spec now extends the shared capability model in `formal/shared/ControllerLifecycleSpec.tla`,
with per-controller family/capability metadata assigned in `formal/controller_manifest.tsv`.
Those capabilities now include convergence-style contracts for paginated lookup, collection equivalence,
whole-list resubmission, and best-effort cleanup where the controller behavior warrants them.
