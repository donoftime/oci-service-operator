#!/usr/bin/env python3

from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
MANIFEST = ROOT / "formal" / "controller_manifest.tsv"
CONTROLLERS_DIR = ROOT / "formal" / "controllers"


def tla_set(raw: str) -> str:
    items = [item.strip() for item in raw.split(",") if item.strip()]
    return "{" + ", ".join(f'"{item}"' for item in items) + "}"


def write_file(path: Path, content: str, overwrite: bool = False) -> None:
    if path.exists() and not overwrite:
        return
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="ascii")


SPEC = """------------------------------- MODULE spec -------------------------------
EXTENDS ControllerLifecycleSpec

=============================================================================
"""


def build_cfg(kind: str, family: str, retryable: str, active: str, failed: str, has_secret: str,
              capabilities: str) -> str:
    return f"""SPECIFICATION Spec
CHECK_DEADLOCK TRUE
CONSTANTS
    ControllerName = "{kind}"
    Family = "{family}"
    RetryableStates = {tla_set(retryable)}
    ActiveStates = {tla_set(active)}
    FailedStates = {tla_set(failed)}
    HasSecret = {has_secret}
    Capabilities = {tla_set(capabilities)}
INVARIANTS
    ControllerMetadataInvariant
    TypeInvariant
    SuccessRequiresActiveInvariant
    RetryableRequiresRequeueInvariant
    DeleteRequiresResourceGoneInvariant
    MutationUsesBoundIDInvariant
    DeleteRequiresConfirmationInvariant
    DeleteSubmittedKeepsFinalizerInvariant
    ConfirmedDeleteRemovesResourceInvariant
    BindByIDUsesSpecInvariant
    ResolvedNameUsesResolvedIDInvariant
    LaterPageResolutionUsesResolvedIDInvariant
    SupportedDriftRequiresUpdateInvariant
    MatchingStateSkipsUpdateInvariant
    CollectionDifferenceRequiresUpdateInvariant
    MatchingCollectionSkipsUpdateInvariant
    WholeListConvergesAfterUpdateInvariant
    SecretRequiresUsableStateInvariant
    SecretWriteFailuresBlockSuccessInvariant
    SecretDeleteFailuresBlockCompletionInvariant
    MissingSecretAllowsDeleteInvariant
    BestEffortCleanupKeepsSuccessInvariant
    CleanupTargetsStayEligibleInvariant
"""


def build_logic_gaps(kind: str, capabilities: str) -> str:
    return f"""# Logic Gaps

- This controller uses the shared capability scaffold for `{kind}` with `{capabilities or "no additional"}`
  capability metadata.
- Record controller-specific TLC counterexamples, failing property tests, and code fixes here as they are confirmed.
"""


def build_agents(kind: str, capabilities: str) -> str:
    return f"""# {kind}

- Source of truth: `spec.tla` and `spec.cfg`
- Shared contracts: `../../shared/ControllerCoreContract.tla`, `../../shared/NameResolutionContract.tla`,
  `../../shared/ListResolutionContract.tla`, `../../shared/DriftAwareUpdateContract.tla`,
  `../../shared/CollectionEquivalenceContract.tla`, `../../shared/WholeListConvergenceContract.tla`,
  `../../shared/BestEffortCleanupContract.tla`, `../../shared/SecretSideEffectContract.tla`
- Diagram source: `diagrams/lifecycle.puml`
- Known gaps and fix history: `logic-gaps.md`
- Capabilities: `{capabilities}`

## Verified Properties

- `ControllerMetadataInvariant`
- `TypeInvariant`
- `SuccessRequiresActiveInvariant`
- `RetryableRequiresRequeueInvariant`
- `DeleteRequiresResourceGoneInvariant`
- `MutationUsesBoundIDInvariant`
- `DeleteRequiresConfirmationInvariant`
- `DeleteSubmittedKeepsFinalizerInvariant`
- `ConfirmedDeleteRemovesResourceInvariant`
- `BindByIDUsesSpecInvariant`
- `ResolvedNameUsesResolvedIDInvariant`
- `LaterPageResolutionUsesResolvedIDInvariant`
- `SupportedDriftRequiresUpdateInvariant`
- `MatchingStateSkipsUpdateInvariant`
- `CollectionDifferenceRequiresUpdateInvariant`
- `MatchingCollectionSkipsUpdateInvariant`
- `WholeListConvergesAfterUpdateInvariant`
- `SecretRequiresUsableStateInvariant`
- `SecretWriteFailuresBlockSuccessInvariant`
- `SecretDeleteFailuresBlockCompletionInvariant`
- `MissingSecretAllowsDeleteInvariant`
- `BestEffortCleanupKeepsSuccessInvariant`
- `CleanupTargetsStayEligibleInvariant`

## Notes

- This file is the controller-local knowledge log for formal verification work.
- Update it with controller-specific counterexamples, linked Go property tests, and the final code fixes.
"""


def build_puml(kind: str, retryable: str, active: str, failed: str) -> str:
    retryable_states = ", ".join(item.strip() for item in retryable.split(",") if item.strip())
    active_states = ", ".join(item.strip() for item in active.split(",") if item.strip())
    failed_states = ", ".join(item.strip() for item in failed.split(",") if item.strip())
    return f"""@startuml
title {kind} Formal Lifecycle

[*] --> Observed
Observed --> Retryable : {retryable_states}
Observed --> Active : {active_states}
Observed --> Failed : {failed_states}
Active --> DeletePending : delete requested
Retryable --> DeletePending : delete requested
Failed --> DeletePending : delete requested
DeletePending --> Deleted : finalizer removed after delete

state Retryable
state Active
state Failed
state DeletePending
state Deleted
@enduml
"""


def main() -> None:
    rows = MANIFEST.read_text(encoding="ascii").strip().splitlines()
    for row in rows[1:]:
        slug, kind, family, retryable, active, failed, has_secret, capabilities = row.split("\t")
        controller_dir = CONTROLLERS_DIR / slug
        write_file(controller_dir / "spec.tla", SPEC, overwrite=True)
        write_file(controller_dir / "spec.cfg",
                   build_cfg(kind, family, retryable, active, failed, has_secret, capabilities), overwrite=True)
        write_file(controller_dir / "logic-gaps.md", build_logic_gaps(kind, capabilities))
        write_file(controller_dir / "AGENTS.md", build_agents(kind, capabilities), overwrite=True)
        write_file(controller_dir / "diagrams" / "lifecycle.puml", build_puml(kind, retryable, active, failed))


if __name__ == "__main__":
    main()
