import fs from "fs-extra";
import * as path from "node:path";
import * as YAML from "yaml";
import { sha256String } from "./hash.js";

export interface Packet {
  schema_version: string;
  packet_id: string;
  task_id: string;
  type: "claim";
  created_at: string;
  previous_packet_id: string | null;
  payload: Record<string, unknown>;
  payload_hash: string;
}

function canonicalJson(obj: Record<string, unknown>): string {
  return JSON.stringify(obj, Object.keys(obj).sort());
}

export function computePayloadHash(payload: Record<string, unknown>): string {
  return sha256String(canonicalJson(payload));
}

function generatePacketId(taskId: string): string {
  const ts = new Date().toISOString().replace(/[:.]/g, "-");
  return `packet-${ts}-${taskId}`;
}

export async function createPacket(
  cardPath: string,
  packetsDir = ".x-harness/packets"
): Promise<{ packet: Packet; filePath: string }> {
  await fs.ensureDir(packetsDir);

  const cardContent = await fs.readFile(cardPath, "utf-8");
  const card = YAML.parse(cardContent) as Record<string, unknown>;

  const taskId = String(card.task_id ?? "unknown");
  const payload: Record<string, unknown> = {
    card_path: path.resolve(cardPath),
    card_content: card,
  };

  const payloadHash = computePayloadHash(payload);

  // Find existing packets for this task to set previous_packet_id
  const existing = await listPacketsForTask(taskId, packetsDir);
  const previousPacketId =
    existing.length > 0 ? existing[existing.length - 1].packet_id : null;

  const packet: Packet = {
    schema_version: "1",
    packet_id: generatePacketId(taskId),
    task_id: taskId,
    type: "claim",
    created_at: new Date().toISOString(),
    previous_packet_id: previousPacketId,
    payload,
    payload_hash: payloadHash,
  };

  const fileName = `${packet.packet_id}.yaml`;
  const filePath = path.join(packetsDir, fileName);

  if (await fs.pathExists(filePath)) {
    throw new Error(`Packet file already exists: ${filePath}`);
  }

  await fs.writeFile(filePath, YAML.stringify(packet));

  return { packet, filePath };
}

export async function readPacket(
  packetId: string,
  packetsDir = ".x-harness/packets"
): Promise<Packet | null> {
  const filePath = path.join(packetsDir, `${packetId}.yaml`);
  if (!(await fs.pathExists(filePath))) return null;
  const content = await fs.readFile(filePath, "utf-8");
  return YAML.parse(content) as Packet;
}

function isPacketLike(obj: unknown): obj is Packet {
  const p = obj as Record<string, unknown> | null | undefined;
  if (!p) return false;
  return (
    typeof p.packet_id === "string" &&
    typeof p.task_id === "string" &&
    typeof p.payload_hash === "string" &&
    p.payload !== undefined
  );
}

export async function listPacketsForTask(
  taskId: string,
  packetsDir = ".x-harness/packets"
): Promise<Packet[]> {
  if (!(await fs.pathExists(packetsDir))) return [];
  const entries = await fs.readdir(packetsDir);
  const packets: Packet[] = [];

  for (const entry of entries) {
    if (!entry.endsWith(".yaml") && !entry.endsWith(".yml")) continue;
    const content = await fs.readFile(path.join(packetsDir, entry), "utf-8");
    const parsed = YAML.parse(content);
    if (!isPacketLike(parsed)) continue;
    const packet = parsed as Packet;
    if (packet.task_id === taskId) {
      packets.push(packet);
    }
  }

  // Sort by created_at ascending to form chain order
  packets.sort(
    (a, b) =>
      new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  );
  return packets;
}

export interface ChainVerificationResult {
  valid: boolean;
  packetsChecked: number;
  firstBrokenIndex: number | null;
  firstBrokenPacketId: string | null;
  expectedHash: string | null;
  actualHash: string | null;
  reason: string | null;
}

export function verifyPacketChain(packets: Packet[]): ChainVerificationResult {
  if (packets.length === 0) {
    return {
      valid: true,
      packetsChecked: 0,
      firstBrokenIndex: null,
      firstBrokenPacketId: null,
      expectedHash: null,
      actualHash: null,
      reason: null,
    };
  }

  // Build index by packet_id
  const byId = new Map<string, Packet>();
  for (const p of packets) {
    byId.set(p.packet_id, p);
  }

  // Check for forks: each packet should be referenced by at most one next packet
  const childCount = new Map<string, number>();
  for (const p of packets) {
    if (p.previous_packet_id) {
      childCount.set(
        p.previous_packet_id,
        (childCount.get(p.previous_packet_id) ?? 0) + 1
      );
    }
  }
  for (const [parentId, count] of childCount.entries()) {
    if (count > 1) {
      return {
        valid: false,
        packetsChecked: packets.length,
        firstBrokenIndex: null,
        firstBrokenPacketId: parentId,
        expectedHash: null,
        actualHash: null,
        reason: `fork detected: packet ${parentId} has ${count} children`,
      };
    }
  }

  // Verify each packet's hash and parent existence
  for (let i = 0; i < packets.length; i++) {
    const p = packets[i];
    const expectedHash = computePayloadHash(p.payload);
    if (p.payload_hash !== expectedHash) {
      return {
        valid: false,
        packetsChecked: i + 1,
        firstBrokenIndex: i,
        firstBrokenPacketId: p.packet_id,
        expectedHash,
        actualHash: p.payload_hash,
        reason: `payload hash mismatch for packet ${p.packet_id}`,
      };
    }

    if (p.previous_packet_id) {
      const parent = byId.get(p.previous_packet_id);
      if (!parent) {
        return {
          valid: false,
          packetsChecked: i + 1,
          firstBrokenIndex: i,
          firstBrokenPacketId: p.packet_id,
          expectedHash: null,
          actualHash: null,
          reason: `missing parent packet ${p.previous_packet_id} for packet ${p.packet_id}`,
        };
      }
    }
  }

  // Detect cycles and orphans by traversing from every unvisited packet
  const visited = new Set<string>();
  let traversalIndex = 0;

  for (const start of packets) {
    if (visited.has(start.packet_id)) continue;

    const pathVisited = new Set<string>();
    let current: Packet | undefined = start;

    while (current) {
      if (pathVisited.has(current.packet_id)) {
        return {
          valid: false,
          packetsChecked: traversalIndex + 1,
          firstBrokenIndex: null,
          firstBrokenPacketId: current.packet_id,
          expectedHash: null,
          actualHash: null,
          reason: `cycle detected at packet ${current.packet_id}`,
        };
      }

      if (visited.has(current.packet_id)) {
        // Joined an already-validated chain; stop
        break;
      }

      pathVisited.add(current.packet_id);
      visited.add(current.packet_id);
      traversalIndex++;

      // Move to next packet in chain
      const next = packets.find(
        (p) => p.previous_packet_id === current!.packet_id
      );
      current = next;
    }
  }

  // After traversal, all packets should be visited
  if (visited.size !== packets.length) {
    const orphan = packets.find((p) => !visited.has(p.packet_id));
    return {
      valid: false,
      packetsChecked: visited.size,
      firstBrokenIndex: null,
      firstBrokenPacketId: orphan?.packet_id ?? null,
      expectedHash: null,
      actualHash: null,
      reason: `orphan packet ${orphan?.packet_id} not reachable from root`,
    };
  }

  return {
    valid: true,
    packetsChecked: packets.length,
    firstBrokenIndex: null,
    firstBrokenPacketId: null,
    expectedHash: null,
    actualHash: null,
    reason: null,
  };
}
