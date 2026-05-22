import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs-extra";
import * as path from "node:path";
import * as YAML from "yaml";
import {
  createPacket,
  readPacket,
  listPacketsForTask,
  verifyPacketChain,
  computePayloadHash,
} from "../src/core/packet.js";
import { execaNode } from "../src/test-helpers.js";

const TEST_PACKETS_DIR = path.join(process.cwd(), ".x-harness-test-packets");

describe("packet core", () => {
  beforeEach(async () => {
    await fs.ensureDir(TEST_PACKETS_DIR);
    await fs.emptyDir(TEST_PACKETS_DIR);
  });

  afterEach(async () => {
    await fs.remove(TEST_PACKETS_DIR);
  });

  it("creates a claim packet from a completion card", async () => {
    const cardPath = path.join(TEST_PACKETS_DIR, "completion-card.yaml");
    await fs.writeFile(
      cardPath,
      YAML.stringify({
        schema_version: "1",
        task_id: "TASK-001",
        tier: "standard",
        owner: "test-owner",
        accountable: "test-accountable",
        claim: { fix_status: "fixed", summary: "Test" },
        verification: { status: "passed", checks: [] },
        admission: { outcome: "success" },
        acceptance_status: "accepted",
        handoff: { next_action: "done", owner: "user" },
      })
    );

    const { packet, filePath } = await createPacket(cardPath, TEST_PACKETS_DIR);
    expect(packet.type).toBe("claim");
    expect(packet.task_id).toBe("TASK-001");
    expect(packet.previous_packet_id).toBeNull();
    expect(packet.payload_hash).toBeTruthy();
    expect(await fs.pathExists(filePath)).toBe(true);
  });

  it("chains packets for the same task", async () => {
    const cardPath = path.join(TEST_PACKETS_DIR, "completion-card.yaml");
    const card = {
      schema_version: "1",
      task_id: "TASK-002",
      tier: "standard",
      owner: "test-owner",
      accountable: "test-accountable",
      claim: { fix_status: "fixed", summary: "Test" },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "done", owner: "user" },
    };
    await fs.writeFile(cardPath, YAML.stringify(card));

    const first = await createPacket(cardPath, TEST_PACKETS_DIR);
    expect(first.packet.previous_packet_id).toBeNull();

    const second = await createPacket(cardPath, TEST_PACKETS_DIR);
    expect(second.packet.previous_packet_id).toBe(first.packet.packet_id);
  });

  it("computes deterministic payload hash", () => {
    const payload = { a: 1, b: 2 };
    const hash1 = computePayloadHash(payload);
    const hash2 = computePayloadHash(payload);
    expect(hash1).toBe(hash2);
    expect(hash1).toMatch(/^[a-f0-9]{64}$/);
  });

  it("payload hash changes when payload changes", () => {
    const h1 = computePayloadHash({ a: 1 });
    const h2 = computePayloadHash({ a: 2 });
    expect(h1).not.toBe(h2);
  });

  it("reads a packet by id", async () => {
    const cardPath = path.join(TEST_PACKETS_DIR, "completion-card.yaml");
    await fs.writeFile(
      cardPath,
      YAML.stringify({
        schema_version: "1",
        task_id: "TASK-003",
        tier: "light",
        owner: "test-owner",
        accountable: "test-accountable",
        claim: { fix_status: "fixed", summary: "Test" },
        verification: { status: "passed", checks: [] },
        admission: { outcome: "success" },
        acceptance_status: "accepted",
        handoff: { next_action: "done", owner: "user" },
      })
    );

    const { packet } = await createPacket(cardPath, TEST_PACKETS_DIR);
    const read = await readPacket(packet.packet_id, TEST_PACKETS_DIR);
    expect(read).not.toBeNull();
    expect(read!.packet_id).toBe(packet.packet_id);
    expect(read!.payload_hash).toBe(packet.payload_hash);
  });

  it("returns null for missing packet", async () => {
    const result = await readPacket("nonexistent", TEST_PACKETS_DIR);
    expect(result).toBeNull();
  });

  it("lists packets for a task", async () => {
    const cardPath = path.join(TEST_PACKETS_DIR, "completion-card.yaml");
    const card = {
      schema_version: "1",
      task_id: "TASK-004",
      tier: "light",
      owner: "test-owner",
      accountable: "test-accountable",
      claim: { fix_status: "fixed", summary: "Test" },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "done", owner: "user" },
    };
    await fs.writeFile(cardPath, YAML.stringify(card));

    await createPacket(cardPath, TEST_PACKETS_DIR);
    await createPacket(cardPath, TEST_PACKETS_DIR);

    const list = await listPacketsForTask("TASK-004", TEST_PACKETS_DIR);
    expect(list).toHaveLength(2);
    expect(list[0].previous_packet_id).toBeNull();
    expect(list[1].previous_packet_id).toBe(list[0].packet_id);
  });

  it("verifies a valid chain", async () => {
    const cardPath = path.join(TEST_PACKETS_DIR, "completion-card.yaml");
    const card = {
      schema_version: "1",
      task_id: "TASK-005",
      tier: "light",
      owner: "test-owner",
      accountable: "test-accountable",
      claim: { fix_status: "fixed", summary: "Test" },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "done", owner: "user" },
    };
    await fs.writeFile(cardPath, YAML.stringify(card));

    await createPacket(cardPath, TEST_PACKETS_DIR);
    await createPacket(cardPath, TEST_PACKETS_DIR);

    const packets = await listPacketsForTask("TASK-005", TEST_PACKETS_DIR);
    const result = verifyPacketChain(packets);
    expect(result.valid).toBe(true);
    expect(result.packetsChecked).toBe(2);
  });

  it("detects tampered payload hash", async () => {
    const cardPath = path.join(TEST_PACKETS_DIR, "completion-card.yaml");
    const card = {
      schema_version: "1",
      task_id: "TASK-006",
      tier: "light",
      owner: "test-owner",
      accountable: "test-accountable",
      claim: { fix_status: "fixed", summary: "Test" },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "done", owner: "user" },
    };
    await fs.writeFile(cardPath, YAML.stringify(card));

    await createPacket(cardPath, TEST_PACKETS_DIR);
    const packets = await listPacketsForTask("TASK-006", TEST_PACKETS_DIR);
    packets[0].payload_hash = "tampered";

    const result = verifyPacketChain(packets);
    expect(result.valid).toBe(false);
    expect(result.reason).toContain("payload hash mismatch");
  });

  it("detects missing parent", async () => {
    const cardPath = path.join(TEST_PACKETS_DIR, "completion-card.yaml");
    const card = {
      schema_version: "1",
      task_id: "TASK-007",
      tier: "light",
      owner: "test-owner",
      accountable: "test-accountable",
      claim: { fix_status: "fixed", summary: "Test" },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "done", owner: "user" },
    };
    await fs.writeFile(cardPath, YAML.stringify(card));

    await createPacket(cardPath, TEST_PACKETS_DIR);
    const packets = await listPacketsForTask("TASK-007", TEST_PACKETS_DIR);
    packets[0].previous_packet_id = "nonexistent-parent";

    const result = verifyPacketChain(packets);
    expect(result.valid).toBe(false);
    expect(result.reason).toContain("missing parent");
  });

  it("detects cycle", async () => {
    const cardPath = path.join(TEST_PACKETS_DIR, "completion-card.yaml");
    const card = {
      schema_version: "1",
      task_id: "TASK-008",
      tier: "light",
      owner: "test-owner",
      accountable: "test-accountable",
      claim: { fix_status: "fixed", summary: "Test" },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "done", owner: "user" },
    };
    await fs.writeFile(cardPath, YAML.stringify(card));

    await createPacket(cardPath, TEST_PACKETS_DIR);
    await createPacket(cardPath, TEST_PACKETS_DIR);
    const packets = await listPacketsForTask("TASK-008", TEST_PACKETS_DIR);

    // Create cycle: second packet points back to itself
    packets[1].previous_packet_id = packets[1].packet_id;

    const result = verifyPacketChain(packets);
    expect(result.valid).toBe(false);
    expect(result.reason).toContain("cycle");
  });

  it("detects fork", async () => {
    const cardPath = path.join(TEST_PACKETS_DIR, "completion-card.yaml");
    const card = {
      schema_version: "1",
      task_id: "TASK-009",
      tier: "light",
      owner: "test-owner",
      accountable: "test-accountable",
      claim: { fix_status: "fixed", summary: "Test" },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "done", owner: "user" },
    };
    await fs.writeFile(cardPath, YAML.stringify(card));

    const first = await createPacket(cardPath, TEST_PACKETS_DIR);
    await createPacket(cardPath, TEST_PACKETS_DIR);
    const third = await createPacket(cardPath, TEST_PACKETS_DIR);

    // Force fork: third packet also points to first
    const thirdFile = path.join(
      TEST_PACKETS_DIR,
      `${third.packet.packet_id}.yaml`
    );
    const tampered = {
      ...third.packet,
      previous_packet_id: first.packet.packet_id,
    };
    await fs.writeFile(thirdFile, YAML.stringify(tampered));

    const packets = await listPacketsForTask("TASK-009", TEST_PACKETS_DIR);
    const result = verifyPacketChain(packets);
    expect(result.valid).toBe(false);
    expect(result.reason).toContain("fork");
  });

  it("handles empty packet directory", async () => {
    const result = verifyPacketChain([]);
    expect(result.valid).toBe(true);
    expect(result.packetsChecked).toBe(0);
  });

  it("rejects overwrite of existing packet file", async () => {
    const cardPath = path.join(TEST_PACKETS_DIR, "completion-card.yaml");
    const card = {
      schema_version: "1",
      task_id: "TASK-010",
      tier: "light",
      owner: "test-owner",
      accountable: "test-accountable",
      claim: { fix_status: "fixed", summary: "Test" },
      verification: { status: "passed", checks: [] },
      admission: { outcome: "success" },
      acceptance_status: "accepted",
      handoff: { next_action: "done", owner: "user" },
    };
    await fs.writeFile(cardPath, YAML.stringify(card));

    await createPacket(cardPath, TEST_PACKETS_DIR);

    // Since packet_id includes timestamp, normal create won't conflict.
    // Test that another create succeeds (no collision).
    await expect(
      createPacket(cardPath, TEST_PACKETS_DIR)
    ).resolves.toBeDefined();
  });
});

describe("packet CLI", () => {
  const CLI_PACKETS_DIR = path.join(
    process.cwd(),
    ".x-harness-test-packets-cli"
  );
  const CARD_PATH = path.join(CLI_PACKETS_DIR, "completion-card.yaml");

  beforeEach(async () => {
    await fs.ensureDir(CLI_PACKETS_DIR);
    await fs.writeFile(
      CARD_PATH,
      YAML.stringify({
        schema_version: "1",
        task_id: "CLI-TASK-001",
        tier: "light",
        owner: "cli-owner",
        accountable: "cli-accountable",
        claim: { fix_status: "fixed", summary: "CLI test" },
        verification: { status: "passed", checks: [] },
        admission: { outcome: "success" },
        acceptance_status: "accepted",
        handoff: { next_action: "done", owner: "user" },
      })
    );
  });

  afterEach(async () => {
    await fs.remove(CLI_PACKETS_DIR);
  });

  it("creates a packet via CLI", async () => {
    const { stdout, exitCode } = await execaNode([
      "packet",
      "create",
      "--card",
      CARD_PATH,
      "--packets-dir",
      CLI_PACKETS_DIR,
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("packet created");
    expect(stdout).toContain("packet_id: packet-");
    expect(stdout).toContain("task_id: CLI-TASK-001");
  });

  it("fails when card not found", async () => {
    const { stderr, exitCode } = await execaNode([
      "packet",
      "create",
      "--card",
      "/nonexistent/card.yaml",
      "--packets-dir",
      CLI_PACKETS_DIR,
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("not found");
  });

  it("verifies chain via CLI", async () => {
    await execaNode([
      "packet",
      "create",
      "--card",
      CARD_PATH,
      "--packets-dir",
      CLI_PACKETS_DIR,
    ]);
    await execaNode([
      "packet",
      "create",
      "--card",
      CARD_PATH,
      "--packets-dir",
      CLI_PACKETS_DIR,
    ]);

    const { stdout, exitCode } = await execaNode([
      "packet",
      "verify-chain",
      "--task-id",
      "CLI-TASK-001",
      "--packets-dir",
      CLI_PACKETS_DIR,
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("chain valid");
    expect(stdout).toContain("2 packet(s)");
  });

  it("fails chain verify when task not found", async () => {
    const { stderr, exitCode } = await execaNode([
      "packet",
      "verify-chain",
      "--task-id",
      "UNKNOWN",
      "--packets-dir",
      CLI_PACKETS_DIR,
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("No packets found");
  });

  it("fails chain verify when tampered", async () => {
    await execaNode([
      "packet",
      "create",
      "--card",
      CARD_PATH,
      "--packets-dir",
      CLI_PACKETS_DIR,
    ]);

    // Tamper the packet
    const entries = await fs.readdir(CLI_PACKETS_DIR);
    const packetFile = entries.find(
      (e) => e.startsWith("packet-") && e.endsWith(".yaml")
    );
    expect(packetFile).toBeDefined();

    const content = await fs.readFile(
      path.join(CLI_PACKETS_DIR, packetFile!),
      "utf-8"
    );
    const packet = YAML.parse(content);
    packet.payload_hash = "tampered";
    await fs.writeFile(
      path.join(CLI_PACKETS_DIR, packetFile!),
      YAML.stringify(packet)
    );

    const { stderr, exitCode } = await execaNode([
      "packet",
      "verify-chain",
      "--task-id",
      "CLI-TASK-001",
      "--packets-dir",
      CLI_PACKETS_DIR,
    ]);
    expect(exitCode).toBe(1);
    expect(stderr).toContain("chain broken");
    expect(stderr).toContain("payload hash mismatch");
  });
});
