# OCI Queue

Source of truth: [spec.tla](spec.tla) and [spec.cfg](spec.cfg).

- Owned code: `pkg/servicemanager/queue/queue_servicemanager.go`
- Shared contract: `formal/shared/BaseReconcilerContract.tla`
- Property tests: `pkg/servicemanager/queue/queue_properties_test.go`
- Diagram: `diagrams/state-machine.puml`
- Verified properties: success only on `ACTIVE`, retryable states requeue, finalizer stays until delete completes, secrets only in usable states
