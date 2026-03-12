# Logic Gaps

- Fixed: create-by-name previously returned unsuccessful without requeue and could stall under the shared reconciler contract.
- Fixed: `CREATING` and `UPDATING` lifecycle states no longer report terminal success.
- Fixed: bind/update paths now preserve the target OCID before issuing updates.
- Fixed: delete errors now propagate, and successful delete completion requires a follow-up `GetOpensearchCluster` proving deletion.
- Fixed: supported OpenSearch drift is now classified into immutable-reject, horizontal resize, vertical resize, and update requests so node-count, node-sizing, security, software-version, and tag drift all reconcile through the appropriate OCI APIs.
- Fixed: audited immutable networking and host-shape fields now fail closed through CEL and reconcile-time validation before any resize or update mutation is submitted.

## Pending Update Surface Audit

### Should Reconcile In Place
- None identified in this pass.

### Should Reject Updates
- None identified in this pass.
