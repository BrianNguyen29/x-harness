import { afterEach, describe, expect, it } from "vitest";
import fs from "fs-extra";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import { execaNode } from "../src/test-helpers.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const passCandidate = path.join(
  repoRoot,
  "tools",
  "experimental",
  "evolve",
  "fixtures",
  "pass-candidate.yaml"
);
const violatingCandidate = path.join(
  repoRoot,
  "tools",
  "experimental",
  "evolve",
  "fixtures",
  "violating-candidate.yaml"
);
const tempFiles: string[] = [];

afterEach(async () => {
  for (const file of tempFiles.splice(0)) {
    await fs.remove(file);
  }
});

function changeRequestOut(name: string): string {
  const out = path.join(
    repoRoot,
    ".x-harness",
    "evolution",
    "change-requests",
    `${name}-${Date.now()}-${Math.random().toString(16).slice(2)}.md`
  );
  tempFiles.push(out);
  return out;
}

describe("evolve command", () => {
  it("evaluates the disabled evolution budget", async () => {
    const { stdout, exitCode } = await execaNode([
      "evolve",
      "evaluate",
      "--root",
      repoRoot,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(true);
    expect(output.status).toBe("disabled");
    expect(output.admission_authority).toBe(false);
  });

  it("passes a safe candidate through constitution-check", async () => {
    const { stdout, exitCode } = await execaNode([
      "evolve",
      "constitution-check",
      "--root",
      repoRoot,
      "--candidate",
      passCandidate,
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(true);
    expect(output.status).toBe("passed");
    expect(output.candidate_id).toBe("cand_pass_001");
    expect(output.admission_authority).toBe(false);
  });

  it("fails a candidate that violates protected invariants", async () => {
    const { stdout, exitCode } = await execaNode([
      "evolve",
      "constitution-check",
      "--root",
      repoRoot,
      "--candidate",
      violatingCandidate,
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(false);
    expect(output.status).toBe("failed");
    expect(output.violations.join("\n")).toContain("disable_mutation_guard");
    expect(output.violations.join("\n")).toContain("false_accept");
  });

  it("writes proposal requests without editing source files", async () => {
    const out = changeRequestOut("proposal");
    const before = await fs.readFile(
      path.join(repoRoot, "policies", "admission.yaml"),
      "utf-8"
    );

    const { stdout, exitCode } = await execaNode([
      "evolve",
      "propose",
      "--root",
      repoRoot,
      "--component",
      "admission_policy",
      "--out",
      out,
      "--json",
    ]);

    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.status).toBe("written");
    expect(output.path).toBe(out);
    expect(await fs.pathExists(out)).toBe(true);
    expect(await fs.readFile(out, "utf-8")).toContain(
      "admission_authority: false"
    );
    const after = await fs.readFile(
      path.join(repoRoot, "policies", "admission.yaml"),
      "utf-8"
    );
    expect(after).toBe(before);
  });

  it("writes promotion and rollback requests without granting authority", async () => {
    const promotion = changeRequestOut("promotion");
    const rollback = changeRequestOut("rollback");

    const promoted = await execaNode([
      "evolve",
      "promote",
      "--root",
      repoRoot,
      "--candidate",
      passCandidate,
      "--out",
      promotion,
      "--json",
    ]);
    expect(promoted.exitCode).toBe(0);
    expect(JSON.parse(promoted.stdout).admission_authority).toBe(false);
    expect(await fs.readFile(promotion, "utf-8")).toContain(
      "This file is a change request only."
    );

    const rolledBack = await execaNode([
      "evolve",
      "rollback",
      "--root",
      repoRoot,
      "--candidate",
      passCandidate,
      "--out",
      rollback,
      "--json",
    ]);
    expect(rolledBack.exitCode).toBe(0);
    expect(JSON.parse(rolledBack.stdout).admission_authority).toBe(false);
    expect(await fs.readFile(rollback, "utf-8")).toContain(
      "This file is a change request only."
    );
  });

  it("blocks promotion for a violating candidate", async () => {
    const { stdout, exitCode } = await execaNode([
      "evolve",
      "promote",
      "--root",
      repoRoot,
      "--candidate",
      violatingCandidate,
      "--out",
      changeRequestOut("blocked-promotion"),
      "--json",
    ]);
    expect(exitCode).toBe(1);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(false);
    expect(output.status).toBe("failed");
  });

  it("rejects change request output outside the managed request directory", async () => {
    const { stderr, exitCode } = await execaNode([
      "evolve",
      "propose",
      "--root",
      repoRoot,
      "--component",
      "admission_policy",
      "--out",
      path.join(repoRoot, "policies", "admission.yaml"),
      "--json",
    ]);
    expect(exitCode).toBe(1);
    expect(stderr).toContain(
      "evolution change requests must be written under .x-harness/evolution/change-requests"
    );
  });
});
