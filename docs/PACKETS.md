# Packets Design Specification

> Status: design/spec only. Implementation deferred per oracle WAIT verdict.

## Overview

Packets are immutable, self-describing evidence containers that x-harness agents can generate, exchange, and verify. This document specifies the packet architecture, lifecycle, and verification rules.

## Design principles

1. **Immutability** — once created, a packet's content hash is fixed.
2. **Self-description** — every packet declares its type, schema version, and creator.
3. **Offline-first** — no network calls required to create or verify packets.
4. **Lightweight** — stays within the x-harness file-first, no-daemon constraint.

## Packet types

| Type | Purpose | Status |
|------|---------|--------|
| `claim` | Agent's completion claim | defined in schema |
| `evidence` | Verification artifacts and scope | defined in schema |
| `cg_packet` | Context-grounding snapshot | **design only** |
| `procedure_pack` | Reusable task procedure | **design only** |
| `recovery_packet` | Recovery route candidate | **design only** |

## Packet structure (proposed)

```yaml
packet:
  id: "pkt-<uuid>"
  type: "claim" | "evidence" | "cg_packet" | "procedure_pack" | "recovery_packet"
  schema_version: 1
  created_at: "2026-01-01T00:00:00Z"
  creator: "agent-name"
  payload: {}
  payload_hash: "sha256:..."
  previous_packet_id: "pkt-<uuid>" | null
  signature: null  # reserved for future use
```

## Verification rules (proposed)

1. `payload_hash` must match `sha256(JSON.stringify(payload))`.
2. `previous_packet_id` must reference an existing packet if non-null.
3. `schema_version` must be supported by the verifier.
4. `type` must be in the known packet type registry.

## Packet chain (proposed)

Packets can form a chain for audit trails:

```
packet-1 (null previous) -> packet-2 (previous=packet-1) -> packet-3
```

A verify-chain command would validate:
- Hash integrity of each packet
- Previous linkage consistency
- No gaps or forks

## Deferred implementation note (P4.6)

The following are **explicitly NOT implemented** in the current phase:

- `create` / `verify-chain` CLI subcommands for packets
- `cg_packet` generation
- `procedure_pack` authoring tools
- `recovery_packet` generation beyond playbook suggestions
- Packet signatures

These remain design-only until the oracle grants a GO verdict for P4.6.

## Relation to existing features

- **Trace hash chain (P4.7)** — implements a lightweight chain in `events.jsonl` using `previous_hash` and `event_hash`. This validates the chain concept without full packet infrastructure.
- **Recovery playbook (P4.5)** — generates review-required recovery candidates in Markdown. Does not emit `recovery_packet` YAML yet.
- **Completion card** — the existing completion card is conceptually a `claim` + `evidence` packet bundle.

## Constraints maintained

- No database / server / daemon
- No network calls
- File-first, offline-only
- Canonical tiers: light, standard, deep
