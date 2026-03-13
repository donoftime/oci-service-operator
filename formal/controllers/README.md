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

Controller-local diagrams are generated from the shared manifest and the metadata shards under
`formal/controller_diagrams/`. Each controller now carries:

- `diagrams/activity.puml`
- `diagrams/sequence.puml`
- `diagrams/state-machine.puml`

The shared explanations that should not be duplicated across all 25 controllers live under
`formal/shared/diagrams/`.
