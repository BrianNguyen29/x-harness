import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import fs from "fs-extra";
import YAML from "yaml";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

describe("add command", () => {
  it("creates a claim file with default name", async () => {
    const outPath = path.join(__dirname, "claim-test.yaml");
    try {
      const { stdout, exitCode } = await execaNode([
        "add",
        "claim",
        "fix_status=fixed,summary=did work",
        "--out",
        outPath,
      ]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("Added claim ->");
      expect(await fs.pathExists(outPath)).toBe(true);
      const content = await fs.readFile(outPath, "utf-8");
      const data = YAML.parse(content);
      expect(data.fix_status).toBe("fixed");
      expect(data.summary).toBe("did work");
      expect(data.id).toContain("CLAIM-");
      expect(data.created_at).toBeDefined();
    } finally {
      await fs.remove(outPath);
    }
  });

  it("creates an evidence file", async () => {
    const outPath = path.join(__dirname, "evidence-test.yaml");
    try {
      const { stdout, exitCode } = await execaNode([
        "add",
        "evidence",
        "owner=alice",
        "--out",
        outPath,
      ]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("Added evidence ->");
      const content = await fs.readFile(outPath, "utf-8");
      const data = YAML.parse(content);
      expect(data.owner).toBe("alice");
      expect(data.id).toContain("EVIDENCE-");
    } finally {
      await fs.remove(outPath);
    }
  });

  it("creates a completion-card file", async () => {
    const outPath = path.join(__dirname, "completion-card-test.yaml");
    try {
      const { stdout, exitCode } = await execaNode([
        "add",
        "completion-card",
        "task_id=T1,tier=light",
        "--out",
        outPath,
      ]);
      expect(exitCode).toBe(0);
      expect(stdout).toContain("Added completion-card ->");
      const content = await fs.readFile(outPath, "utf-8");
      const data = YAML.parse(content);
      expect(data.task_id).toBe("T1");
      expect(data.tier).toBe("light");
    } finally {
      await fs.remove(outPath);
    }
  });

  it("handles values with colons and newlines via YAML quoting", async () => {
    const outPath = path.join(__dirname, "edge-test.yaml");
    try {
      const { exitCode } = await execaNode([
        "add",
        "claim",
        "description=foo: bar\nbaz: qux",
        "--out",
        outPath,
      ]);
      expect(exitCode).toBe(0);
      const content = await fs.readFile(outPath, "utf-8");
      const data = YAML.parse(content);
      expect(data.description).toBe("foo: bar\nbaz: qux");
    } finally {
      await fs.remove(outPath);
    }
  });
});
