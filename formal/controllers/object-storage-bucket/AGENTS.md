# Object Storage Bucket

Source of truth: [spec.tla](spec.tla) and [spec.cfg](spec.cfg).

- Owned code: `pkg/servicemanager/objectstorage/objectstorage_servicemanager.go`
- Shared contract: `formal/shared/BaseReconcilerContract.tla`
- Property tests: `pkg/servicemanager/objectstorage/objectstorage_properties_test.go`
- Diagram: `diagrams/state-machine.puml`
- Verified properties: success only on usable states, delete keeps the finalizer until the bucket is gone, secrets only exist in usable states
- Go-side delete coverage now includes spec-ID fallback and finalizer completion when the backing Secret is already absent.
