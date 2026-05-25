import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import fs from "fs-extra";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import {
  checkStaleness,
  getSourceOfTruthFiles,
} from "../src/core/staleness.js";
import { generateManagedBlock } from "../src/core/context.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

describe("staleness module", () => {
  it("getSourceOfTruthFiles returns expected files", () => {
    const files = getSourceOfTruthFiles();
    expect(files).toContain("X_HARNESS.md");
    expect(files).toContain("policies/admission.yaml");
    expect(files).toContain("policies/recovery.yaml");
    expect(files).toContain("policies/intake.yaml");
    expect(files).toContain("schemas/completion-card.schema.json");
  });
});

describe("staleness command", () => {
  it("reports missing managed block as error", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-staleness-"));
    try {
      await fs.writeFile(path.join(tmpDir, "AGENTS.md"), "# Empty\n", "utf-8");
      const result = await checkStaleness(tmpDir);
      expect(result.stale).toBe(true);
      expect(
        result.findings.some((f) => f.type === "missing_managed_block")
      ).toBe(true);
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("reports stale hash as error", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-staleness-"));
    try {
      const agentsPath = path.join(tmpDir, "AGENTS.md");
      await fs.writeFile(
        agentsPath,
        `# Agent Contract

<!-- BEGIN X-HARNESS MANAGED CONTEXT -->
<!-- generated-by: x-harness -->
<!-- generated-at: 2025-01-01T00:00:00.000Z -->
<!-- context-hash: deadbeefdeadbeef -->

# x-harness Canonical Context

- Completion is admitted, not claimed.

<!-- END X-HARNESS MANAGED CONTEXT -->
`,
        "utf-8"
      );
      const result = await checkStaleness(tmpDir);
      expect(result.stale).toBe(true);
      expect(result.findings.some((f) => f.type === "stale_hash")).toBe(true);
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("reports tampered managed block body even when hash is unchanged", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-staleness-"));
    try {
      const agentsPath = path.join(tmpDir, "AGENTS.md");
      const block = generateManagedBlock().replace(
        "Completion is admitted, not claimed.",
        "Completion is self-admitted."
      );
      await fs.writeFile(agentsPath, `# Agent Contract\n\n${block}\n`, "utf-8");
      const result = await checkStaleness(tmpDir);
      expect(result.stale).toBe(true);
      expect(result.findings.some((f) => f.type === "stale_hash")).toBe(true);
      expect(result.findings.some((f) => f.message.includes("body"))).toBe(
        true
      );
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("reports missing linked files as warn", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-staleness-"));
    try {
      const agentsPath = path.join(tmpDir, "AGENTS.md");
      await fs.writeFile(
        agentsPath,
        `# Agent Contract

<!-- BEGIN X-HARNESS MANAGED CONTEXT -->
<!-- generated-by: x-harness -->
<!-- generated-at: ${new Date().toISOString()} -->
<!-- context-hash: ${"0".repeat(16)} -->

# x-harness Canonical Context

- Completion is admitted, not claimed.

<!-- END X-HARNESS MANAGED CONTEXT -->
`,
        "utf-8"
      );
      const result = await checkStaleness(tmpDir);
      expect(
        result.findings.some((f) => f.type === "missing_linked_file")
      ).toBe(true);
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("context staleness subcommand outputs JSON", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-staleness-"));
    try {
      await fs.writeFile(path.join(tmpDir, "AGENTS.md"), "# Empty\n", "utf-8");
      const { stdout, exitCode } = await execaNode([
        "context",
        "staleness",
        "--output-json",
        "--staleness-root",
        tmpDir,
      ]);
      expect(exitCode).toBe(1);
      const parsed = JSON.parse(stdout);
      expect(parsed).toHaveProperty("stale", true);
      expect(parsed).toHaveProperty("findings");
      expect(parsed).toHaveProperty("source_of_truth_files");
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("context --check validates managed block", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-staleness-"));
    try {
      await fs.writeFile(path.join(tmpDir, "AGENTS.md"), "# Empty\n", "utf-8");
      const { stderr, exitCode } = await execaNode([
        "context",
        "--check",
        "--root",
        tmpDir,
      ]);
      expect(exitCode).toBe(1);
      expect(stderr).toContain("missing");
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("context --check --json outputs JSON with missing_linked_files", async () => {
    const tmpDir = mkdtempSync(path.join(tmpdir(), "x-harness-staleness-"));
    try {
      const agentsPath = path.join(tmpDir, "AGENTS.md");
      // Create a stale managed block with invalid hash
      await fs.writeFile(
        agentsPath,
        `# Agent Contract

<!-- BEGIN X-HARNESS MANAGED CONTEXT -->
<!-- generated-by: x-harness -->
<!-- generated-at: ${new Date().toISOString()} -->
<!-- context-hash: deadbeefdeadbeef -->

# x-harness Canonical Context

- Completion is admitted, not claimed.

<!-- END X-HARNESS MANAGED CONTEXT -->
`,
        "utf-8"
      );
      const { stdout, exitCode } = await execaNode([
        "context",
        "--check",
        "--json",
        "--root",
        tmpDir,
      ]);
      // Exit code 1 because hash is stale
      expect(exitCode).toBe(1);
      const parsed = JSON.parse(stdout);
      expect(parsed).toHaveProperty("valid", false);
      expect(parsed.note).toContain("stale");
      expect(parsed).toHaveProperty("missing_linked_files");
      expect(Array.isArray(parsed.missing_linked_files)).toBe(true);
    } finally {
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});
