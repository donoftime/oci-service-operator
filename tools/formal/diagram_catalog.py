#!/usr/bin/env python3

from __future__ import annotations

import csv
import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[2]
FORMAL_DIR = ROOT / "formal"
CONTROLLERS_DIR = FORMAL_DIR / "controllers"
METADATA_DIR = FORMAL_DIR / "controller_diagrams"
SHARED_DIAGRAMS_DIR = FORMAL_DIR / "shared" / "diagrams"

CONTROLLER_DIAGRAM_BASENAMES = ("activity", "sequence", "state-machine")
SHARED_DIAGRAM_BASENAMES = (
    "shared-reconcile-activity",
    "shared-resolution-sequence",
    "shared-delete-sequence",
    "shared-controller-state-machine",
    "shared-legend",
)
LEGACY_SHARED_DIAGRAM_BASENAMES = (
    "controller-phase-model",
    "controller-contract-map",
)

DEFAULT_DISPLAY_NAMES = {
    "api-gateway": "API Gateway",
    "api-gateway-deployment": "API Gateway Deployment",
    "autonomous-databases": "Autonomous Databases",
    "compute-instance": "Compute Instance",
    "container-instance": "Container Instance",
    "dataflow-application": "Data Flow Application",
    "functions-application": "Functions Application",
    "functions-function": "Functions Function",
    "mysql-dbsystem": "MySQL DB System",
    "nosql-database": "NoSQL Database",
    "object-storage-bucket": "Object Storage Bucket",
    "oci-drg": "OCI DRG",
    "oci-internet-gateway": "OCI Internet Gateway",
    "oci-nat-gateway": "OCI NAT Gateway",
    "oci-network-security-group": "OCI Network Security Group",
    "oci-queue": "OCI Queue",
    "oci-route-table": "OCI Route Table",
    "oci-security-list": "OCI Security List",
    "oci-service-gateway": "OCI Service Gateway",
    "oci-subnet": "OCI Subnet",
    "oci-vcn": "OCI VCN",
    "open-search-cluster": "Open Search Cluster",
    "postgres-db-system": "Postgres DB System",
    "redis-cluster": "Redis Cluster",
    "stream": "Stream",
}

BASE_INVARIANTS = {
    "ControllerMetadataInvariant",
    "TypeInvariant",
    "SuccessRequiresActiveInvariant",
    "RetryableRequiresRequeueInvariant",
    "DeleteRequiresResourceGoneInvariant",
    "MutationUsesBoundIDInvariant",
    "DeleteRequiresConfirmationInvariant",
    "DeleteSubmittedKeepsFinalizerInvariant",
    "ConfirmedDeleteRemovesResourceInvariant",
    "BindByIDUsesSpecInvariant",
    "ResolvedNameUsesResolvedIDInvariant",
    "LaterPageResolutionUsesResolvedIDInvariant",
    "SupportedDriftRequiresUpdateInvariant",
    "MatchingStateSkipsUpdateInvariant",
    "CollectionDifferenceRequiresUpdateInvariant",
    "MatchingCollectionSkipsUpdateInvariant",
    "WholeListConvergesAfterUpdateInvariant",
    "SecretRequiresUsableStateInvariant",
    "SecretWriteFailuresBlockSuccessInvariant",
    "SecretDeleteFailuresBlockCompletionInvariant",
    "MissingSecretAllowsDeleteInvariant",
    "BestEffortCleanupKeepsSuccessInvariant",
    "CleanupTargetsStayEligibleInvariant",
}

CAPABILITY_FEATURES = {
    "resolve_by_name": "resolve_by_name",
    "paginated_resolution": "paginated_lookup",
    "collection_equivalence": "semantic_collection_diff",
    "whole_list_convergence": "whole_list_resubmission",
    "best_effort_cleanup": "best_effort_cleanup",
    "secret_write": "secret_sync",
    "secret_delete": "delete_cleanup_blocked",
}

KNOWN_FEATURES = {
    "resolve_by_name",
    "paginated_lookup",
    "semantic_collection_diff",
    "whole_list_resubmission",
    "best_effort_cleanup",
    "secret_sync",
    "delete_cleanup_blocked",
    "status_binding",
    "move_compartment",
    "delete_work_request",
    "record_delete_work_request",
    "retryable_read_failures",
    "namespace_resolution",
    "composite_id_validation",
    "secret_owner_guard",
    "secret_endpoint_required",
    "reject_update",
    "resize_horizontal",
    "resize_vertical",
    "duplicate_cleanup",
    "collection_without_contract",
    "inactive_ready_state",
}


@dataclass(frozen=True)
class ControllerDiagramModel:
    slug: str
    kind: str
    family: str
    display_name: str
    archetype: str
    retryable_states: tuple[str, ...]
    active_states: tuple[str, ...]
    failed_states: tuple[str, ...]
    capabilities: frozenset[str]
    features: frozenset[str]
    extra_invariants: tuple[str, ...]
    update_surface: tuple[str, ...]
    ordered_steps: tuple[str, ...]
    reject_paths: tuple[str, ...]
    delete_steps: tuple[str, ...]
    boundary_notes: tuple[str, ...]
    sequence_notes: tuple[str, ...]


def parse_csv_set(raw: str) -> list[str]:
    return [item.strip() for item in raw.split(",") if item.strip()]


def parse_extra_invariants(slug: str) -> list[str]:
    cfg_path = CONTROLLERS_DIR / slug / "spec.cfg"
    extra: list[str] = []
    in_invariants = False
    for raw_line in cfg_path.read_text(encoding="ascii").splitlines():
        stripped = raw_line.strip()
        if stripped == "INVARIANTS":
            in_invariants = True
            continue
        if not in_invariants or not stripped or not raw_line.startswith("    "):
            continue
        if stripped not in BASE_INVARIANTS:
            extra.append(stripped)
    return extra


def load_manifest() -> dict[str, dict[str, Any]]:
    manifest_path = FORMAL_DIR / "controller_manifest.tsv"
    with manifest_path.open(encoding="ascii", newline="") as handle:
        rows = csv.DictReader(handle, delimiter="\t")
        return {row["slug"]: row for row in rows}


def _load_metadata_file(path: Path) -> dict[str, Any]:
    try:
        payload = json.loads(path.read_text(encoding="ascii"))
    except json.JSONDecodeError as exc:
        raise ValueError(f"{path} is not valid JSON-compatible YAML: {exc}") from exc
    if not isinstance(payload, dict) or "controllers" not in payload or not isinstance(payload["controllers"], dict):
        raise ValueError(f"{path} must contain a top-level object with a 'controllers' mapping")
    return payload["controllers"]


def load_metadata() -> dict[str, dict[str, Any]]:
    if not METADATA_DIR.exists():
        raise ValueError(f"missing metadata directory: {METADATA_DIR}")
    merged: dict[str, dict[str, Any]] = {}
    for path in sorted(METADATA_DIR.glob("*.yaml")):
        controllers = _load_metadata_file(path)
        for slug, meta in controllers.items():
            if slug in merged:
                raise ValueError(f"controller {slug} is defined more than once in metadata shards")
            if not isinstance(meta, dict):
                raise ValueError(f"{path} controller {slug} metadata must be an object")
            merged[slug] = meta
    return merged


def build_models() -> list[ControllerDiagramModel]:
    manifest = load_manifest()
    metadata = load_metadata()

    missing = sorted(set(manifest) - set(metadata))
    extra = sorted(set(metadata) - set(manifest))
    if missing:
        raise ValueError(f"missing diagram metadata for controllers: {', '.join(missing)}")
    if extra:
        raise ValueError(f"metadata contains unknown controllers: {', '.join(extra)}")

    models: list[ControllerDiagramModel] = []
    for slug in sorted(manifest):
        row = manifest[slug]
        meta = metadata[slug]
        capabilities = frozenset(parse_csv_set(row["capabilities"]))
        extra_invariants = tuple(parse_extra_invariants(slug))

        features = {CAPABILITY_FEATURES[cap] for cap in capabilities if cap in CAPABILITY_FEATURES}
        features.update(meta.get("features", []))
        if "StatusPresentUsesStatusInvariant" in extra_invariants:
            features.add("status_binding")
        if meta.get("reject_paths"):
            features.add("reject_update")
        unknown_features = sorted(set(features) - KNOWN_FEATURES)
        if unknown_features:
            raise ValueError(f"controller {slug} uses unknown diagram features: {', '.join(unknown_features)}")

        models.append(
            ControllerDiagramModel(
                slug=slug,
                kind=row["kind"],
                family=row["family"],
                display_name=meta.get("display_name", DEFAULT_DISPLAY_NAMES.get(slug, slug)),
                archetype=meta["archetype"],
                retryable_states=tuple(parse_csv_set(row["retryable_states"])),
                active_states=tuple(parse_csv_set(row["active_states"])),
                failed_states=tuple(parse_csv_set(row["failed_states"])),
                capabilities=capabilities,
                features=frozenset(features),
                extra_invariants=extra_invariants,
                update_surface=tuple(meta.get("update_surface", [])),
                ordered_steps=tuple(meta.get("ordered_steps", [])),
                reject_paths=tuple(meta.get("reject_paths", [])),
                delete_steps=tuple(meta.get("delete_steps", [])),
                boundary_notes=tuple(meta.get("boundary_notes", [])),
                sequence_notes=tuple(meta.get("sequence_notes", [])),
            )
        )
    return models
