# Controller Diagram Metadata

This directory contains the controller-local metadata that drives the generated
PlantUML sources under `formal/controllers/<slug>/diagrams/`.

Each `.yaml` file is intentionally written as JSON-compatible YAML so the
generator can parse it with Python's standard library.

## Schema

Top-level structure:

```json
{
  "controllers": {
    "<slug>": {
      "archetype": "resolved-drift-delete",
      "update_surface": ["field one", "field two"],
      "ordered_steps": ["ordered reconcile step"],
      "reject_paths": ["unsupported drift"],
      "delete_steps": ["delete confirmation step"],
      "boundary_notes": ["accepted modeling boundary"],
      "features": ["move_compartment"],
      "sequence_notes": ["extra sequence note"]
    }
  }
}
```

## Field Notes

- `archetype`: grouping used by the shared legend and batching strategy.
- `update_surface`: mutable surface that the controller intentionally reconciles.
- `ordered_steps`: ordered reconcile steps that should be explicit in the activity
  and sequence diagrams.
- `reject_paths`: immutable or unsupported drift that should fail before mutation.
- `delete_steps`: delete confirmation and cleanup steps after delete submission.
- `boundary_notes`: important behavior that stays outside the formal guarantee or
  diverges from the simplified capability model.
- `features`: extra generator flags beyond the manifest capabilities. See
  `tools/formal/diagram_catalog.py` for the supported values.
- `sequence_notes`: short notes that are useful in the sequence diagram but do
  not belong in the main message flow.
