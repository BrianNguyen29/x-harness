# Packets

Packets are immutable, self-describing evidence containers that x-harness agents can generate, exchange, and verify.

## Design principles

1. **Immutability** — once created, a packet's content hash is fixed.
2. **Self-description** — every packet declares its type, schema version, and creator.
3. **Offline-first** — no network calls required to create or verify packets.
4. **Lightweight** — stays within the x-harness file-first, no-daemon constraint.

## Packet types

| Type | Purpose |
|------|---------|
| `claim` | Agent's completion claim |

The `claim` type is the only packet type supported by the current CLI. Other types (`evidence`, `cg_packet`, `procedure_pack`, `recovery_packet`) are defined in the schema but not supported by current commands.

## Packet structure

```yaml
packet:
  id: "pkt-<uuid>"
  type: "claim"
  schema_version: 1
  created_at: "2026-01-01T00:00:00Z"
  creator: "agent-name"
  payload: {}
  payload_hash: "sha256:..."
  previous_packet_id: "pkt-<uuid>" | null
  signature: null
```

## Verification rules

1. `payload_hash` must match `sha256(JSON.stringify(payload))`.
2. `previous_packet_id` must reference an existing packet if non-null.
3. `schema_version` must be supported by the verifier.
4. `type` must be in the known packet type registry.

## Packet chain

Packets can form a chain for audit trails:

```
packet-1 (null previous) -> packet-2 (previous=packet-1) -> packet-3
```

`packet verify-chain` validates:
- Hash integrity of each packet
- Previous linkage consistency
- No gaps or forks

## Implementation guardrails

The following guardrails are enforced in the current implementation:

- **Claim packet only** — `type: "claim"` is the only supported packet type.
- **No git flags** — `packet create` does not auto-commit or auto-add.
- **No admission/verify/trace integration** — packets are standalone artifacts.
- **Flat directory** — all packets live in `.x-harness/packets/` (no subdirectories).
- **Canonical JSON payload hash** — `payload_hash` is computed from sorted-key canonical JSON.
- **Immutable by default** — attempting to overwrite an existing packet file fails.

### Supported CLI commands

```bash
xh packet create --card completion-card.yaml
xh packet verify-chain --task-id <task-id>
```

## Constraints maintained

- No database / server / daemon
- No network calls
- File-first, offline-only
- Canonical tiers: light, standard, deep
