#!/usr/bin/env python3

from __future__ import annotations

import argparse
import textwrap
from pathlib import Path

from diagram_catalog import (
    CONTROLLER_DIAGRAM_BASENAMES,
    CONTROLLERS_DIR,
    LEGACY_SHARED_DIAGRAM_BASENAMES,
    SHARED_DIAGRAM_BASENAMES,
    SHARED_DIAGRAMS_DIR,
    ControllerDiagramModel,
    build_models,
)


def wrap(text: str, width: int = 56) -> list[str]:
    return textwrap.wrap(text, width=width, subsequent_indent="  ")


def join_or_none(items: tuple[str, ...]) -> str:
    return ", ".join(items) if items else "none"


def bullet_block(title: str, items: tuple[str, ...], width: int = 58) -> list[str]:
    if not items:
        return [f"{title}: none"]
    lines = [f"{title}:"]
    for item in items:
        wrapped = wrap(item, width=width)
        if not wrapped:
            continue
        lines.append(f"- {wrapped[0]}")
        for line in wrapped[1:]:
            lines.append(f"  {line}")
    return lines


def state_label(token: str) -> str:
    labels = {
        "resolve_by_name": "ResolveByName",
        "paginated_lookup": "PaginatedLookup",
        "namespace_resolution": "ResolveNamespace",
        "composite_id_validation": "ValidateCompositeID",
        "move_compartment": "MoveCompartment",
        "semantic_collection_diff": "CompareCollection",
        "whole_list_resubmission": "ResubmitWholeList",
        "resize_horizontal": "HorizontalResize",
        "resize_vertical": "VerticalResize",
        "duplicate_cleanup": "CleanupDuplicates",
        "best_effort_cleanup": "BestEffortCleanup",
        "secret_sync": "SyncSecret",
        "delete_work_request": "DeleteWorkRequestPending",
        "record_delete_work_request": "DeleteWorkRequestRecorded",
        "delete_cleanup_blocked": "DeleteCleanupBlocked",
        "reject_update": "RejectUnsupportedDrift",
    }
    return labels[token]


def pre_ready_chain(model: ControllerDiagramModel) -> list[str]:
    chain: list[str] = []
    if "resolve_by_name" in model.features:
        chain.append(state_label("resolve_by_name"))
    if "paginated_lookup" in model.features:
        chain.append(state_label("paginated_lookup"))
    if "namespace_resolution" in model.features:
        chain.append(state_label("namespace_resolution"))
    if "composite_id_validation" in model.features:
        chain.append(state_label("composite_id_validation"))
    return chain


def needs_ready_evaluation(model: ControllerDiagramModel) -> bool:
    return bool(
        model.update_surface
        or model.reject_paths
        or {
            "move_compartment",
            "semantic_collection_diff",
            "whole_list_resubmission",
            "best_effort_cleanup",
            "duplicate_cleanup",
            "secret_sync",
            "resize_horizontal",
            "resize_vertical",
            "retryable_read_failures",
        }
        & model.features
    )


def ready_mutation_states(model: ControllerDiagramModel) -> list[str]:
    chain: list[str] = []
    if "move_compartment" in model.features:
        chain.append(state_label("move_compartment"))
    if "semantic_collection_diff" in model.features:
        chain.append(state_label("semantic_collection_diff"))
    if "whole_list_resubmission" in model.features:
        chain.append(state_label("whole_list_resubmission"))
    if "resize_horizontal" in model.features:
        chain.append(state_label("resize_horizontal"))
    if "resize_vertical" in model.features:
        chain.append(state_label("resize_vertical"))
    if model.update_surface:
        chain.append("ApplyUpdate")
    if "duplicate_cleanup" in model.features:
        chain.append(state_label("duplicate_cleanup"))
    elif "best_effort_cleanup" in model.features:
        chain.append(state_label("best_effort_cleanup"))
    return chain


def delete_chain(model: ControllerDiagramModel) -> list[str]:
    chain = ["DeletePending"]
    if "record_delete_work_request" in model.features:
        chain.append(state_label("record_delete_work_request"))
    if "delete_work_request" in model.features:
        chain.append(state_label("delete_work_request"))
    if "delete_cleanup_blocked" in model.features:
        chain.append(state_label("delete_cleanup_blocked"))
    chain.append("Deleted")
    return chain


def controller_activity_theme(title: str) -> list[str]:
    return [
        "@startuml",
        f"title {title}",
        "skinparam shadowing false",
        "skinparam BackgroundColor #FFFFFF",
        "skinparam ArrowColor #334155",
        "skinparam defaultTextAlignment left",
        "skinparam activity {",
        "  BackgroundColor #F8FAFC",
        "  BorderColor #475569",
        "  FontColor #0F172A",
        "  DiamondBackgroundColor #E2E8F0",
        "  DiamondBorderColor #475569",
        "  StartColor #0F766E",
        "  EndColor #7F1D1D",
        "}",
    ]


def controller_sequence_theme(title: str) -> list[str]:
    return [
        "@startuml",
        f"title {title}",
        "autonumber",
        "skinparam shadowing false",
        "skinparam BackgroundColor #FFFFFF",
        "skinparam ArrowColor #334155",
        "skinparam defaultTextAlignment left",
        "skinparam sequence {",
        "  ParticipantBackgroundColor #F8FAFC",
        "  ParticipantBorderColor #475569",
        "  LifeLineBorderColor #94A3B8",
        "  LifeLineBackgroundColor #FFFFFF",
        "  GroupBorderColor #475569",
        "  GroupBackgroundColor #F8FAFC",
        "  ActorBackgroundColor #E0F2FE",
        "  ActorBorderColor #0F766E",
        "}",
    ]


def controller_state_theme(title: str) -> list[str]:
    return [
        "@startuml",
        f"title {title}",
        "left to right direction",
        "hide empty description",
        "skinparam shadowing false",
        "skinparam linetype ortho",
        "skinparam roundcorner 12",
        "skinparam BackgroundColor #FFFFFF",
        "skinparam defaultTextAlignment left",
        "skinparam state {",
        "  BorderColor #475569",
        "  FontColor #0F172A",
        "  BackgroundColor #F8FAFC",
        "}",
        "skinparam note {",
        "  BorderColor #B45309",
        "  BackgroundColor #FFF7ED",
        "  FontColor #0F172A",
        "}",
    ]


def render_activity(model: ControllerDiagramModel) -> str:
    lines = controller_activity_theme(f"{model.display_name} Reconcile Activity")
    lines.extend(
        [
            "start",
            "",
            'partition "Observe and Bind" {',
            "  :Read CR spec, status OCID, and delete intent;",
        ]
    )
    if "status_binding" in model.features:
        lines.append("  :Keep status-bound OCID authoritative for later update or delete paths;")
    if "resolve_by_name" in model.features:
        lines.append('  if ("Tracked or explicit OCID present?") then (yes)')
        lines.append("    :Get the OCI resource by known identifier;")
        lines.append("  else (no)")
        lines.append("    :Resolve an existing OCI resource by display name;")
        if "paginated_lookup" in model.features:
            lines.append("    :Continue list pagination until a match or exhaustion;")
        lines.append("    :Persist the resolved or created OCID back into status;")
        lines.append("  endif")
    else:
        lines.append("  :Bind the OCI resource through explicit identifiers only;")
    if "namespace_resolution" in model.features:
        lines.append("  :Resolve and persist the Object Storage namespace before mutation or delete;")
    if "composite_id_validation" in model.features:
        lines.append("  :Validate the composite bucket identifier before delete or update;")
    lines.extend(
        [
            "}",
            "",
            'if ("Delete requested?") then (yes)',
            '  partition "Delete" {',
            f"    :Submit OCI delete for {model.display_name};",
        ]
    )
    for step in model.delete_steps:
        lines.append(f"    :{step};")
    if "secret_owner_guard" in model.features:
        lines.append("    :Only delete Secrets that are owned by this controller;")
    if "delete_cleanup_blocked" in model.features:
        lines.append('    if ("Owned Secret cleanup succeeds?") then (yes)')
        lines.append("      :Remove the finalizer after OCI deletion is confirmed;")
        lines.append("    else (no)")
        lines.append("      :Stay blocked until Secret cleanup succeeds or is absent;")
        lines.append("    endif")
    else:
        lines.append("    :Remove the finalizer after OCI deletion is confirmed;")
    lines.extend(["  }", "  stop", "else (no)"])
    lines.extend(
        [
            '  partition "Lifecycle Classification" {',
            '    if ("OCI state in retryable set?") then (yes)',
            "      :Request requeue and keep the finalizer;",
            "      stop",
            "    endif",
            '    if ("OCI state in failed set?") then (yes)',
            "      :Return an unsuccessful terminal reconcile result;",
            "      stop",
            "    endif",
            "  }",
            "",
            '  partition "Ready and Drift Handling" {',
            "    :Compare live OCI state with the supported drift surface;",
        ]
    )
    if model.reject_paths:
        lines.extend(
            [
                '    if ("Unsupported or immutable drift detected?") then (yes)',
                "      :Reject the change before any OCI mutation;",
                "      stop",
                "    endif",
            ]
        )
    if "retryable_read_failures" in model.features:
        lines.append("    :Treat transient OCI read failures as retryable and requeue;")
    if "resize_horizontal" in model.features or "resize_vertical" in model.features:
        lines.append("    :Classify supported drift into horizontal resize, vertical resize, or general update;")
    for step in model.ordered_steps:
        lines.append(f"    :{step};")
    if model.update_surface:
        lines.extend(
            [
                '    if ("Supported drift detected?") then (yes)',
                "      :Apply only the supported in-place update surface;",
                "    else (no)",
                "      :Skip the no-op mutation path;",
                "    endif",
            ]
        )
    else:
        lines.append("    :No supported in-place update surface is modeled for this controller;")
    if "secret_endpoint_required" in model.features:
        lines.append("    :Require the live endpoint before writing Secret data;")
    if "secret_sync" in model.features:
        lines.extend(
            [
                '    if ("Secret sync succeeds?") then (yes)',
                "      :Return success for the usable active state;",
                "    else (no)",
                "      :Block successful completion until Secret sync succeeds;",
                "    endif",
            ]
        )
    else:
        lines.append("    :Return success for the usable active state;")
    lines.extend(["  }", "endif"])

    note_lines = []
    note_lines.extend(bullet_block("Archetype", (model.archetype,)))
    note_lines.extend(bullet_block("Retryable OCI states", model.retryable_states))
    note_lines.extend(bullet_block("Active OCI states", model.active_states))
    note_lines.extend(bullet_block("Failed OCI states", model.failed_states))
    note_lines.extend(bullet_block("Update surface", model.update_surface))
    note_lines.extend(bullet_block("Reject before mutate", model.reject_paths))
    note_lines.extend(bullet_block("Boundary notes", model.boundary_notes))
    if model.extra_invariants:
        note_lines.extend(bullet_block("Controller-local invariants", model.extra_invariants))

    lines.extend(["", "floating note right", *note_lines, "end note", "", "@enduml", ""])
    return "\n".join(lines)


def render_sequence(model: ControllerDiagramModel) -> str:
    lines = controller_sequence_theme(f"{model.display_name} Reconcile Sequence")
    lines.extend(
        [
            'actor "Controller" as Controller',
            'participant "Service Manager" as ServiceManager',
            'database "OCI" as OCI',
            'database "Kubernetes API" as K8s',
        ]
    )
    if "delete_work_request" in model.features or "record_delete_work_request" in model.features:
        lines.append('collections "Work Request" as WorkRequest')
    lines.extend(
        [
            "",
            "Controller -> ServiceManager: reconcile desired spec and live status",
            "ServiceManager -> K8s: read CR status and finalizer state",
            "",
            "group Lookup and bind",
        ]
    )
    if "resolve_by_name" in model.features:
        lines.extend(
            [
                "  alt tracked or explicit OCID already exists",
                "    ServiceManager -> OCI: get the current resource by known identifier",
                "  else no OCID is bound yet",
                "    ServiceManager -> OCI: list resources by display name",
            ]
        )
        if "paginated_lookup" in model.features:
            lines.extend(
                [
                    "    loop later pages until a match or exhaustion",
                    "      ServiceManager -> OCI: fetch the next list page",
                    "    end",
                ]
            )
        lines.extend(
            [
                "    alt existing resource found",
                "      ServiceManager -> K8s: persist the resolved OCID in status",
                "    else no existing resource found",
                "      ServiceManager -> OCI: create the OCI resource",
                "      ServiceManager -> K8s: persist the created OCID in status",
                "    end",
                "  end",
            ]
        )
    else:
        lines.append("  ServiceManager -> OCI: bind or create through explicit identifiers only")
    if "namespace_resolution" in model.features:
        lines.append("  ServiceManager -> OCI: resolve and persist the bucket namespace")
    if "composite_id_validation" in model.features:
        lines.append("  ServiceManager -> ServiceManager: validate the composite identifier before mutation")
    lines.extend(["end", ""])

    lines.append("alt delete requested")
    lines.append("  group Delete")
    lines.append("    ServiceManager -> OCI: submit OCI delete")
    if "record_delete_work_request" in model.features:
        lines.append("    OCI --> ServiceManager: return a delete work-request ID")
        lines.append("    ServiceManager -> K8s: record delete progress while retaining the finalizer")
    if "delete_work_request" in model.features:
        if "record_delete_work_request" not in model.features:
            lines.append("    OCI --> ServiceManager: return a delete work-request ID")
        lines.append("    ServiceManager -> WorkRequest: wait for the matching delete work request to succeed")
    for step in model.delete_steps:
        target = "OCI"
        lowered = step.lower()
        if "secret" in lowered or "wallet" in lowered:
            target = "K8s"
        elif "finalizer" in lowered:
            target = "K8s"
        elif "work request" in lowered:
            target = "WorkRequest"
        elif "record" in lowered and "work-request" in lowered:
            target = "K8s"
        lines.append(f"    ServiceManager -> {target}: {step}")
    if "delete_cleanup_blocked" in model.features:
        lines.append("    alt owned Secret cleanup succeeds")
        lines.append("      ServiceManager -> K8s: remove the finalizer")
        lines.append("    else Secret cleanup fails")
        lines.append("      ServiceManager --> Controller: retain the finalizer and retry")
        lines.append("    end")
    else:
        lines.append("    ServiceManager -> K8s: remove the finalizer after delete confirmation")
    lines.extend(["  end", "else OCI state is retryable", "  ServiceManager --> Controller: requeue required"])
    lines.extend(["else OCI state is failed or terminal", "  ServiceManager --> Controller: unsuccessful terminal reconcile result"])
    lines.extend(["else OCI state is active and usable", "  group Drift handling"])
    if model.update_surface:
        lines.append("    Note over ServiceManager,OCI")
        for line in bullet_block("Supported update surface", model.update_surface, width=68):
            lines.append(f"      {line}")
        if model.reject_paths:
            for line in bullet_block("Reject before mutate", model.reject_paths, width=68):
                lines.append(f"      {line}")
        lines.append("    end note")
    if model.reject_paths:
        lines.extend(
            [
                "    opt unsupported or immutable drift is detected",
                "      ServiceManager --> Controller: reject before OCI mutation",
                "    end",
            ]
        )
    if "retryable_read_failures" in model.features:
        lines.extend(
            [
                "    opt transient OCI read failure occurs",
                "      ServiceManager --> Controller: requeue instead of surfacing a hard failure",
                "    end",
            ]
        )
    if "resize_horizontal" in model.features:
        lines.append("    ServiceManager -> OCI: apply the horizontal resize branch when node count drifts")
    if "resize_vertical" in model.features:
        lines.append("    ServiceManager -> OCI: apply the vertical resize branch when node sizing drifts")
    for step in model.ordered_steps:
        lines.append(f"    ServiceManager -> OCI: {step}")
    if model.update_surface:
        lines.extend(
            [
                "    opt supported drift or collection diff exists",
                "      ServiceManager -> OCI: apply the supported in-place mutation path",
                "    end",
            ]
        )
    if "secret_sync" in model.features:
        if "secret_endpoint_required" in model.features:
            lines.append("    ServiceManager -> OCI: verify the live endpoint needed for Secret generation")
        lines.extend(
            [
                "    ServiceManager -> K8s: upsert the owned Secret for the usable active resource",
                "    alt Secret sync fails",
                "      ServiceManager --> Controller: block success and retry",
                "    end",
            ]
        )
    lines.extend(["  end", "  ServiceManager --> Controller: successful active reconcile", "end"])

    if model.boundary_notes or model.sequence_notes or model.extra_invariants:
        lines.extend(["", "Note over Controller,OCI"])
        for line in bullet_block("Boundary notes", model.boundary_notes, width=72):
            lines.append(f"  {line}")
        for line in bullet_block("Sequence notes", model.sequence_notes, width=72):
            lines.append(f"  {line}")
        if model.extra_invariants:
            for line in bullet_block("Controller-local invariants", model.extra_invariants, width=72):
                lines.append(f"  {line}")
        lines.extend(["end note"])

    lines.extend(["", "@enduml", ""])
    return "\n".join(lines)


def render_state_machine(model: ControllerDiagramModel) -> str:
    lines = controller_state_theme(f"{model.display_name} Reconcile State Machine")
    lines.extend(["[*] --> Observe", "Observe : read spec, status, delete intent, and OCI lifecycle"])

    entry_state = "Observe"
    if "resolve_by_name" in model.features:
        lines.append("Observe --> ResolveByName : status/spec OCID missing")
        entry_state = "ResolveByName"
        if "paginated_lookup" in model.features:
            lines.append("ResolveByName --> PaginatedLookup : continue searching later list pages")
            entry_state = "PaginatedLookup"
    if "namespace_resolution" in model.features:
        lines.append(f"{entry_state} --> ResolveNamespace : namespace is required before mutate or delete")
        entry_state = "ResolveNamespace"
    if "composite_id_validation" in model.features:
        lines.append(f"{entry_state} --> ValidateCompositeID : validate the bucket composite identifier")
        entry_state = "ValidateCompositeID"

    if needs_ready_evaluation(model):
        lines.append(f"{entry_state} --> EvaluateReady : OCI state in {join_or_none(model.active_states)}")
    else:
        lines.append(f"{entry_state} --> Ready : OCI state in {join_or_none(model.active_states)}")
    lines.append(f"{entry_state} --> Retryable : OCI state in {join_or_none(model.retryable_states)}")
    lines.append(f"{entry_state} --> Failed : OCI state in {join_or_none(model.failed_states)}")

    if model.reject_paths:
        lines.append("EvaluateReady --> RejectUnsupportedDrift : unsupported or immutable drift is detected")
        lines.append("RejectUnsupportedDrift --> Ready : wait for the spec or live state to change")

    mutation_states = ready_mutation_states(model)
    if mutation_states:
        first_state = mutation_states[0]
        lines.append(f"EvaluateReady --> {first_state} : continue active reconcile")
        if "move_compartment" in model.features:
            move_target = "ApplyUpdate" if model.update_surface else ("SyncSecret" if "secret_sync" in model.features else "Ready")
            lines.append(f"MoveCompartment --> {move_target} : continue after compartment move")
        if "semantic_collection_diff" in model.features:
            collection_target = "ResubmitWholeList" if "whole_list_resubmission" in model.features else ("ApplyUpdate" if model.update_surface else ("SyncSecret" if "secret_sync" in model.features else "Ready"))
            lines.append(f"CompareCollection --> {collection_target} : semantic collection diff exists")
            lines.append("CompareCollection --> Ready : matching collection skips mutation")
        if "whole_list_resubmission" in model.features:
            convergence_target = "ApplyUpdate" if model.update_surface else ("SyncSecret" if "secret_sync" in model.features else "Ready")
            lines.append(f"ResubmitWholeList --> {convergence_target} : full desired collection is resubmitted")
        if "resize_horizontal" in model.features:
            resize_target = "SyncSecret" if "secret_sync" in model.features else "Ready"
            lines.append(f"HorizontalResize --> {resize_target} : horizontal resize succeeds")
        if "resize_vertical" in model.features:
            resize_target = "SyncSecret" if "secret_sync" in model.features else "Ready"
            lines.append(f"VerticalResize --> {resize_target} : vertical resize succeeds")
        if model.update_surface:
            update_target = "SyncSecret" if "secret_sync" in model.features else "Ready"
            lines.append(f"ApplyUpdate --> {update_target} : supported mutation path completes")
        if "duplicate_cleanup" in model.features:
            cleanup_target = "SyncSecret" if "secret_sync" in model.features else "Ready"
            lines.append(f"CleanupDuplicates --> {cleanup_target} : eligible duplicate cleanup stays non-blocking")
        elif "best_effort_cleanup" in model.features:
            cleanup_target = "SyncSecret" if "secret_sync" in model.features else "Ready"
            lines.append(f"BestEffortCleanup --> {cleanup_target} : cleanup remains non-blocking")
    elif needs_ready_evaluation(model):
        if "secret_sync" in model.features:
            lines.append("EvaluateReady --> SyncSecret : usable active state requires Secret sync")
        else:
            lines.append("EvaluateReady --> Ready : no supported drift remains")

    if "secret_sync" in model.features:
        lines.append("SyncSecret --> SecretBlocked : Secret write fails")
        lines.append("SecretBlocked --> SyncSecret : retry Secret sync")
        lines.append("SyncSecret --> Ready : Secret side effects succeed")

    lines.append("Ready --> Ready : no supported drift remains")
    lines.append("Retryable --> Retryable : OCI remains nonterminal")
    lines.append("Failed --> Failed : OCI remains terminal")

    delete_states = delete_chain(model)
    for source in ("Ready", "Retryable", "Failed"):
        lines.append(f"{source} --> DeletePending : delete requested")
    for earlier, later in zip(delete_states, delete_states[1:]):
        if earlier == "DeleteCleanupBlocked" and later == "Deleted":
            label = "retry Secret cleanup until completion is allowed"
        else:
            label = {
                "DeleteWorkRequestRecorded": "record the returned work-request identifier",
                "DeleteWorkRequestPending": "wait for the matching delete work request",
                "DeleteCleanupBlocked": "owned Secret cleanup fails after OCI delete",
                "Deleted": "OCI deletion is confirmed and the finalizer can be removed",
            }.get(later, "continue delete workflow")
        lines.append(f"{earlier} --> {later} : {label}")
    lines.append("Deleted --> Deleted : terminal stutter")

    note_lines = [
        *bullet_block("Archetype", (model.archetype,)),
        *bullet_block("Update surface", model.update_surface),
        *bullet_block("Reject before mutate", model.reject_paths),
        *bullet_block("Boundary notes", model.boundary_notes),
    ]
    if model.extra_invariants:
        note_lines.extend(bullet_block("Controller-local invariants", model.extra_invariants))
    lines.extend(["", "note right of Ready", *note_lines, "end note"])

    delete_note = [
        *bullet_block("Delete states", tuple(delete_states)),
        *bullet_block("Delete workflow", model.delete_steps),
    ]
    lines.extend(["", "note right of DeletePending", *delete_note, "end note", "", "@enduml", ""])
    return "\n".join(lines)


def render_shared_reconcile_activity() -> str:
    lines = controller_activity_theme("Shared Reconcile Activity")
    lines.extend(
        [
            "start",
            'partition "Observe and Bind" {',
            "  :Read spec, status, finalizer, and delete intent;",
            "  :Prefer bound OCIDs from status or spec when they exist;",
            "  :Resolve by name only when no OCID is available;",
            "}",
            'if ("Delete requested?") then (yes)',
            '  partition "Delete" {',
            "    :Submit OCI delete and retain the finalizer;",
            "    :Confirm that the OCI resource is gone;",
            "    :Optionally clean up owned Secrets;",
            "    :Remove the finalizer only after deletion is confirmed;",
            "  }",
            "  stop",
            "else (no)",
            '  partition "Lifecycle Classification" {',
            '    if ("OCI state is retryable?") then (yes)',
            "      :Request requeue;",
            "      stop",
            "    endif",
            '    if ("OCI state is failed?") then (yes)',
            "      :Return an unsuccessful reconcile result;",
            "      stop",
            "    endif",
            "  }",
            '  partition "Ready and Drift Handling" {',
            "    :Diff the supported update surface against live OCI state;",
            "    :Reject unsupported drift before mutation;",
            "    :Apply supported updates, collection convergence, or cleanup branches;",
            "    :Write Secrets only for usable active resources;",
            "    :Return success for the ready state;",
            "  }",
            "endif",
            "",
            "floating note right",
            "- This shared diagram explains the generic flow every controller diagram specializes.",
            "- Controller-local activity diagrams add the ordered mutation steps, reject surfaces,",
            "  delete confirmation mode, and side-effect branches specific to one controller.",
            "end note",
            "",
            "@enduml",
            "",
        ]
    )
    return "\n".join(lines)


def render_shared_resolution_sequence() -> str:
    lines = controller_sequence_theme("Shared Resolution Sequence")
    lines.extend(
        [
            'actor "Controller" as Controller',
            'participant "Service Manager" as ServiceManager',
            'database "OCI" as OCI',
            'database "Kubernetes API" as K8s',
            "",
            "Controller -> ServiceManager: reconcile desired and observed state",
            "group Identifier selection",
            "  alt status or spec OCID already exists",
            "    ServiceManager -> OCI: Get the resource by known identifier",
            "  else no OCID is bound yet",
            "    ServiceManager -> OCI: List by display name",
            "    loop later pages when pagination is enabled",
            "      ServiceManager -> OCI: Fetch the next page until a match or exhaustion",
            "    end",
            "    alt existing resource found",
            "      ServiceManager -> K8s: Persist the resolved OCID in status",
            "    else no existing resource found",
            "      ServiceManager -> OCI: Create the OCI resource",
            "      ServiceManager -> K8s: Persist the created OCID in status",
            "    end",
            "  end",
            "end",
            "",
            "note over ServiceManager,OCI",
            "- Bind-by-ID keeps spec/status OCIDs authoritative once they exist.",
            "- Name resolution stays on the resolved OCID for later update and delete paths.",
            "- Paginated lookup must keep using the later-page match, not restart from scratch.",
            "end note",
            "",
            "@enduml",
            "",
        ]
    )
    return "\n".join(lines)


def render_shared_delete_sequence() -> str:
    lines = controller_sequence_theme("Shared Delete and Finalizer Sequence")
    lines.extend(
        [
            'actor "Controller" as Controller',
            'participant "Service Manager" as ServiceManager',
            'database "OCI" as OCI',
            'database "Kubernetes API" as K8s',
            'collections "Work Request" as WorkRequest',
            "",
            "Controller -> ServiceManager: reconcile delete request",
            "ServiceManager -> OCI: submit OCI delete",
            "opt delete path returns a work-request ID",
            "  OCI --> ServiceManager: return a work-request identifier",
            "  ServiceManager -> WorkRequest: wait for the matching delete work request when the controller models it",
            "end",
            "loop until the delete is confirmed",
            "  ServiceManager -> OCI: Get the current resource",
            "end",
            "opt controller owns a Secret side effect",
            "  ServiceManager -> K8s: delete the owned Secret only after OCI deletion is confirmed",
            "end",
            "ServiceManager -> K8s: remove the finalizer after deletion confirmation",
            "",
            "note over ServiceManager,K8s",
            "- Finalizers remain until OCI deletion is confirmed.",
            "- Secret cleanup failures block completion when the controller models Secret deletion.",
            "- Missing Secrets may still allow completion for controllers that treat deletion as best effort.",
            "end note",
            "",
            "@enduml",
            "",
        ]
    )
    return "\n".join(lines)


def render_shared_controller_state_machine() -> str:
    lines = controller_state_theme("Shared Controller State Machine")
    lines.extend(
        [
            "[*] --> Observe",
            "Observe : read spec, status, finalizer, and OCI lifecycle",
            "Observe --> ResolveByName : no OCID is bound yet",
            "ResolveByName --> PaginatedLookup : later pages must be searched",
            "PaginatedLookup --> Ready : an active resource is found or created",
            "Observe --> Retryable : OCI state is nonterminal",
            "Observe --> Failed : OCI state is terminal",
            "Ready --> MoveCompartment : compartment drift is modeled",
            "MoveCompartment --> ApplyUpdate : continue with the supported mutation path",
            "Ready --> ApplyUpdate : supported drift exists",
            "ApplyUpdate --> SyncSecret : Secret generation is modeled",
            "ApplyUpdate --> ResubmitWholeList : full desired collections must converge",
            "ApplyUpdate --> BestEffortCleanup : eligible duplicate cleanup is modeled",
            "SyncSecret --> Ready : Secret side effects succeed",
            "ResubmitWholeList --> Ready : the desired collection converges",
            "BestEffortCleanup --> Ready : non-blocking cleanup completes",
            "Ready --> DeletePending : delete requested",
            "Retryable --> DeletePending : delete requested",
            "Failed --> DeletePending : delete requested",
            "DeletePending --> DeleteWorkRequestPending : delete is tracked through a work request",
            "DeletePending --> DeleteCleanupBlocked : Secret cleanup fails after OCI delete",
            "DeletePending --> Deleted : OCI deletion is confirmed",
            "DeleteWorkRequestPending --> Deleted : delete work request succeeds and OCI deletion is confirmed",
            "DeleteCleanupBlocked --> Deleted : cleanup retry succeeds",
            "Deleted --> Deleted : terminal stutter",
            "",
            "note right of Ready",
            "- Controller-local state machines prune or expand these shared states.",
            "- For example: namespace resolution, resize branches, immutable-drift rejects,",
            "  best-effort cleanup, or work-request-backed delete paths.",
            "end note",
            "",
            "@enduml",
            "",
        ]
    )
    return "\n".join(lines)


def render_shared_legend(models: list[ControllerDiagramModel]) -> str:
    counts: dict[str, int] = {}
    for model in models:
        counts[model.archetype] = counts.get(model.archetype, 0) + 1
    archetype_lines = tuple(f"{name}: {count} controller(s)" for name, count in sorted(counts.items()))

    lines = [
        "@startuml",
        "title Shared Diagram Legend",
        "left to right direction",
        "skinparam shadowing false",
        "skinparam BackgroundColor #FFFFFF",
        "skinparam defaultTextAlignment left",
        "skinparam rectangle {",
        "  BorderColor #475569",
        "  FontColor #0F172A",
        "  BackgroundColor #FFFFFF",
        "}",
        'rectangle "Blue lifecycle notes\\n==\\nRetryable, active, and failed OCI state buckets.\\nThese come directly from the formal controller manifest." as lifecycle #E0F2FE',
        'rectangle "Green ready-path boxes\\n==\\nSupported update, collection, resize, cleanup, and Secret branches.\\nThese combine TLA-proved capabilities with controller-local implementation detail." as ready #DCFCE7',
        'rectangle "Amber boundary notes\\n==\\nReject-before-mutate surfaces and accepted modeling boundaries.\\nThese call out controller behavior that is important to readers or explicitly outside the model." as boundary #FEF3C7',
        'rectangle "Orange delete boxes\\n==\\nDelete confirmation, work-request tracking, Secret cleanup, and finalizer discipline." as delete #FFEDD5',
        'rectangle "Archetype batches\\n==\\n'
        + "\\n".join(archetype_lines)
        + '" as archetypes #F8FAFC',
        "lifecycle --> ready",
        "ready --> boundary",
        "ready --> delete",
        "boundary --> archetypes",
        "@enduml",
        "",
    ]
    return "\n".join(lines)


def write_text(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="ascii")


def cleanup_stale_controller_files(model: ControllerDiagramModel) -> None:
    diagram_dir = CONTROLLERS_DIR / model.slug / "diagrams"
    diagram_dir.mkdir(parents=True, exist_ok=True)
    keep = {f"{name}.puml" for name in CONTROLLER_DIAGRAM_BASENAMES} | {f"{name}.svg" for name in CONTROLLER_DIAGRAM_BASENAMES}
    for path in diagram_dir.iterdir():
        if path.name not in keep and path.suffix in {".puml", ".svg"}:
            path.unlink()


def cleanup_stale_shared_files() -> None:
    SHARED_DIAGRAMS_DIR.mkdir(parents=True, exist_ok=True)
    keep = {f"{name}.puml" for name in SHARED_DIAGRAM_BASENAMES} | {f"{name}.svg" for name in SHARED_DIAGRAM_BASENAMES}
    legacy = {f"{name}.puml" for name in LEGACY_SHARED_DIAGRAM_BASENAMES} | {f"{name}.svg" for name in LEGACY_SHARED_DIAGRAM_BASENAMES}
    for path in SHARED_DIAGRAMS_DIR.iterdir():
        if path.name in legacy or (path.suffix in {".puml", ".svg"} and path.name not in keep):
            path.unlink()


def generate_shared(models: list[ControllerDiagramModel]) -> None:
    cleanup_stale_shared_files()
    write_text(SHARED_DIAGRAMS_DIR / "shared-reconcile-activity.puml", render_shared_reconcile_activity())
    write_text(SHARED_DIAGRAMS_DIR / "shared-resolution-sequence.puml", render_shared_resolution_sequence())
    write_text(SHARED_DIAGRAMS_DIR / "shared-delete-sequence.puml", render_shared_delete_sequence())
    write_text(SHARED_DIAGRAMS_DIR / "shared-controller-state-machine.puml", render_shared_controller_state_machine())
    write_text(SHARED_DIAGRAMS_DIR / "shared-legend.puml", render_shared_legend(models))


def generate_controller(model: ControllerDiagramModel) -> None:
    cleanup_stale_controller_files(model)
    diagram_dir = CONTROLLERS_DIR / model.slug / "diagrams"
    write_text(diagram_dir / "activity.puml", render_activity(model))
    write_text(diagram_dir / "sequence.puml", render_sequence(model))
    write_text(diagram_dir / "state-machine.puml", render_state_machine(model))


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate formal PlantUML sources from controller metadata.")
    parser.add_argument("--controller", dest="controller", help="Generate diagrams for only one controller slug.")
    parser.add_argument("--shared-only", action="store_true", help="Generate only shared diagrams.")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    models = build_models()
    model_map = {model.slug: model for model in models}

    if args.controller:
        if args.controller not in model_map:
            raise ValueError(f"unknown controller slug: {args.controller}")
        generate_controller(model_map[args.controller])
        return

    generate_shared(models)
    if args.shared_only:
        return

    for model in models:
        generate_controller(model)


if __name__ == "__main__":
    main()
