# Object Storage Bucket Logic Gaps

- Fixed: namespace resolution no longer mutates `spec.namespace` inside reconcile without persisting it.
- Fixed: connection secrets no longer emit the literal `<region>` placeholder in the endpoint URL.
- Fixed: delete only completes once a follow-up `GetBucket` proves the bucket is gone; malformed composite IDs now fail closed.
- Residual risk: the formal model focuses on lifecycle/finalizer/secret behavior, not full bucket drift coverage.
