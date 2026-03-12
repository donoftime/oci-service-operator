# Logic Gaps

- Fixed: create-by-name previously returned unsuccessful without requeue and could stall under the shared reconciler contract.
- Fixed: `CREATING` and `UPDATING` lifecycle states no longer report terminal success.
- Fixed: bind/update paths now preserve the target OCID before issuing updates.
- Fixed: delete errors now propagate, and successful delete completion requires a follow-up `GetOpensearchCluster` proving deletion.
