# OCI Service Operator for Kubernetes (OSOK)

This is a Kubernetes operator (kubebuilder v3, controller-runtime v0.17) that manages
OCI cloud resources as Kubernetes custom resources.

## Key Facts

- **Language**: Go 1.21
- **Framework**: kubebuilder / controller-runtime
- **OCI SDK**: `github.com/oracle/oci-go-sdk/v65`
- **Build**: `go build ./...`
- **Tests**: `go test ./...`
- **Code generation**: `make generate` (deepcopy) + `make manifests` (CRD yaml)

## Project Layout

```
api/v1beta1/           # CRD type definitions (*_types.go)
controllers/           # Controller reconcilers (*_controller.go)
pkg/servicemanager/    # OCI service manager implementations
  apigateway/          # API Gateway service manager (good reference)
  containerinstance/   # Container Instance (good reference for Compute)
  streams/             # Streams (simple reference pattern)
  mysql/dbsystem/      # MySQL DB System
  ...
main.go                # Controller registration
config/crd/            # Generated CRD manifests
```

## Adding a New Service — Checklist

1. **CRD types** (`api/v1beta1/<service>_types.go`):
   - Define `<Service>Spec`, `<Service>Status`, `<Service>` struct
   - Add `+kubebuilder:object:root=true` and `+kubebuilder:subresource:status`
   - Register with `SchemeBuilder.Register` in `init()`
   - Copy the groupversion_info.go pattern

2. **Service manager** (`pkg/servicemanager/<service>/`):
   - `<service>_serviceclient.go`: OCI client interface + implementations
   - `<service>_servicemanager.go`: implements `OSOKServiceManager` (CreateOrUpdate, Delete, GetCrdStatus)
   - Follow `containerinstance` or `streams` as the reference pattern

3. **Controller** (`controllers/<service>_controller.go`):
   - Copy `controllers/containerinstance_controller.go` exactly
   - Change the resource type references

4. **Register in main.go**:
   - Copy the ContainerInstance registration block, adjust types

5. **Generate**:
   ```bash
   make generate   # Generates zz_generated.deepcopy.go
   make manifests  # Generates config/crd/bases/*.yaml
   ```

6. **Update** `config/crd/kustomization.yaml` to include new CRD yaml

7. **Build and test**:
   ```bash
   go build ./...
   go test ./...
   ```

## Important Patterns

### OCI Client Interface (testability)
Every service manager has an injected OCI client interface field:
```go
type FooServiceManager struct {
    Provider  common.ConfigurationProvider
    ociClient FooClientInterface  // nil = create from Provider
}
func (c *FooServiceManager) getOCIClient() FooClientInterface {
    if c.ociClient != nil { return c.ociClient }
    client, _ := oci_service.NewFooClientWithConfigurationProvider(c.Provider)
    return client
}
```

### Lifecycle State Handling
Always handle non-terminal states with a requeue:
```go
if instance.LifecycleState == "FAILED" {
    // set failed status, return false
} else if instance.LifecycleState == "ACTIVE" {
    // set active status, return true
} else {
    // set provisioning status
    return OSOKResponse{IsSuccessful: false}, fmt.Errorf("waiting for ACTIVE, currently %s", instance.LifecycleState)
}
```

### Conditional OCI Fields
Never send zero-value optional fields. Use conditionals:
```go
if spec.Port != 0 { details.Port = common.Int(spec.Port) }
if spec.Description != "" { details.Description = common.String(spec.Description) }
```

### Secret Generation
After resource is ACTIVE, write endpoint/connection info to a k8s Secret.
See `gateway_secretgeneration.go` or `containerinstance` for the pattern.

## Polecat Work Notes

When implementing a new service:
- You MUST run `make generate && make manifests` after adding types
- You MUST run `go build ./...` to verify compilation
- You MUST commit ALL generated files (zz_generated.deepcopy.go, CRD yaml)
- Do NOT close your bead until you have run `gt done` which pushes your branch
- The refinery merges your branch — you do not push to main directly

### CRITICAL: Verify your commit exists before `gt done`

Before EVER running `gt done`, you MUST run:
```bash
git log origin/main..HEAD --oneline
```
If this shows NOTHING, you have not committed your work. `gt done` will submit
an empty branch to the merge queue and your work will be lost. Fix this by
committing your changes first.

This is the single most common failure mode in this repo. Check it every time.
