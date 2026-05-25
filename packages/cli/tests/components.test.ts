import { describe, expect, it } from "vitest";
import { execaNode } from "../src/test-helpers.js";
import {
  componentPathCoversPattern,
  componentPathMatches,
} from "../src/core/components.js";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));

describe("components command", () => {
  it("validates registry and protected path coverage", async () => {
    const { stdout, exitCode } = await execaNode([
      "components",
      "validate",
      "--json",
      "--root",
      repoRoot,
    ]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(true);
    expect(output.component_count).toBeGreaterThan(0);
    expect(output.protected_paths_checked).toBeGreaterThan(0);
    expect(output.protected_paths_covered).toBe(output.protected_paths_checked);
  });

  it("lists registered components", async () => {
    const { stdout, exitCode } = await execaNode([
      "components",
      "list",
      "--json",
      "--root",
      repoRoot,
    ]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    const ids = output.components.map(
      (component: { id: string }) => component.id
    );
    expect(ids).toContain("admission_policy");
    expect(ids).toContain("component_registry");
  });

  it("explains a component by id", async () => {
    const { stdout, exitCode } = await execaNode([
      "components",
      "explain",
      "--id",
      "admission_policy",
      "--json",
      "--root",
      repoRoot,
    ]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.id).toBe("admission_policy");
    expect(output.paths).toContain("policies/admission.yaml");
  });

  it("maps changed files to components", async () => {
    const { stdout, exitCode } = await execaNode([
      "components",
      "changed",
      "--files",
      "packages/cli/src/core/admission.ts,examples/ci/strict-verify/completion-card.yaml,unknown/file.txt",
      "--json",
      "--root",
      repoRoot,
    ]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    const ids = output.components.map(
      (component: { id: string }) => component.id
    );
    expect(ids).toContain("admission_policy");
    expect(ids).toContain("examples_and_golden");
    expect(output.unregistered_files).toContain("unknown/file.txt");
  });

  it("fails validation when a protected path is not registered", async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-components-"));
    try {
      fs.mkdirSync(path.join(tmpDir, "policies"), { recursive: true });
      fs.mkdirSync(path.join(tmpDir, "components"), { recursive: true });
      fs.writeFileSync(
        path.join(tmpDir, "policies", "authority.yaml"),
        `version: 1
authority_classes:
  human_only:
    description: protected
    examples: []
protected_paths:
  - path: policies/admission.yaml
    authority: human_only
    rationale: admission policy
report_only: true
governance_check:
  behavior: warn
  exit_on_warnings: false
  block_on_violations: false
`,
        "utf-8"
      );
      fs.writeFileSync(
        path.join(tmpDir, "components", "registry.yaml"),
        `version: 1
components:
  - id: docs_only
    kind: docs
    paths:
      - docs/**
    owner: maintainers
    stability: stable
    agent_edit: agent_editable
    tests:
      - npm test
`,
        "utf-8"
      );

      const { stdout, exitCode } = await execaNode([
        "components",
        "validate",
        "--json",
        "--root",
        tmpDir,
      ]);
      expect(exitCode).toBe(1);
      const output = JSON.parse(stdout);
      expect(output.ok).toBe(false);
      expect(output.errors[0]).toContain(
        "protected path is not registered to any component"
      );
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });
});

describe("component path matching", () => {
  it("matches files against component globs", () => {
    expect(componentPathMatches("schemas/**", "schemas/foo.schema.json")).toBe(
      true
    );
    expect(
      componentPathMatches(
        "packages/cli/src/validators/*.ts",
        "packages/cli/src/validators/base.ts"
      )
    ).toBe(true);
    expect(componentPathMatches("templates/**", "docs/README.md")).toBe(false);
  });

  it("checks protected path coverage by broader component patterns", () => {
    expect(componentPathCoversPattern("schemas/**", "schemas/**")).toBe(true);
    expect(
      componentPathCoversPattern(
        "schemas/**",
        "schemas/completion-card.schema.json"
      )
    ).toBe(true);
    expect(
      componentPathCoversPattern("docs/**", "policies/admission.yaml")
    ).toBe(false);
  });
});
