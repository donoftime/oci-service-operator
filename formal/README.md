# Formal Verification Layout

- `formal/shared/` contains reusable TLA+ operators and shared controller contract notes.
- `formal/controllers/<controller-slug>/` contains one controller's source of truth:
  - `spec.tla`
  - `spec.cfg`
  - `logic-gaps.md`
  - `AGENTS.md`
  - `diagrams/*.puml`

The repo-level entrypoints are:

- `make formal-tools`
- `make formal`
- `make formal-<controller-slug>`
- `make diagrams`
- `make diagrams-<controller-slug>`

The shared controller contract is grounded in `pkg/core/reconciler.go`:

- non-terminal OCI states must not be reported as terminal success
- retryable states must explicitly request requeue
- finalizers must remain until deletion is complete
- secrets must only be created for usable resources
