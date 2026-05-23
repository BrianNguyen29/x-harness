import { describe, it, expect } from "vitest";
import * as path from "node:path";
import * as fs from "node:fs";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));

describe("adapter contract", () => {
  interface AdapterCheck {
    name: string;
    path: string;
    requiredFiles: string[];
    existenceOnly?: boolean;
  }

  const adapters: AdapterCheck[] = [
    {
      name: "generic",
      path: "adapters/generic",
      requiredFiles: ["README.md", "AGENTS.md"],
    },
    {
      name: "claude-code",
      path: "adapters/claude-code",
      requiredFiles: [
        "README.md",
        "CLAUDE.md",
        "agents/implementation-worker.md",
        "agents/admission-verifier.md",
      ],
    },
    {
      name: "cursor",
      path: "adapters/cursor",
      requiredFiles: ["README.md", "rules/x-harness.mdc"],
    },
    {
      name: "opencode",
      path: "adapters/opencode",
      requiredFiles: ["README.md", "verify-agent.md"],
    },
    {
      name: "antigravity",
      path: "adapters/antigravity",
      requiredFiles: [
        "README.md",
        "rules/x-harness.md",
        "workflows/x-harness-implementation.md",
        "workflows/x-harness-verify.md",
      ],
    },
  ];

  for (const adapter of adapters) {
    describe(adapter.name, () => {
      for (const file of adapter.requiredFiles) {
        const fullPath = path.join(repoRoot, adapter.path, file);
        it(`has ${file}`, () => {
          expect(fs.existsSync(fullPath), `${fullPath} should exist`).toBe(
            true
          );
        });

        if (!adapter.existenceOnly) {
          it(`${file} is non-empty`, () => {
            const content = fs.readFileSync(fullPath, "utf-8");
            expect(content.trim().length).toBeGreaterThan(0);
          });
        }
      }

      it("adapter directory exists", () => {
        const adapterDir = path.join(repoRoot, adapter.path);
        expect(fs.existsSync(adapterDir)).toBe(true);
        const entries = fs.readdirSync(adapterDir);
        expect(entries.length).toBeGreaterThan(0);
      });
    });
  }

  describe("GitHub Actions adapter", () => {
    const actionPath = path.join(
      repoRoot,
      "examples",
      "actions",
      "x-harness-verify",
      "action.yml"
    );

    it("action.yml exists", () => {
      expect(fs.existsSync(actionPath)).toBe(true);
    });

    it("action.yml has required inputs", () => {
      const content = fs.readFileSync(actionPath, "utf-8");
      expect(content).toContain("card-path");
      expect(content).toContain("trace-dir");
      expect(content).toContain("trace");
    });

    it("action.yml uses composite run steps", () => {
      const content = fs.readFileSync(actionPath, "utf-8");
      expect(content).toContain("using: composite");
      expect(content).toContain("steps:");
    });
  });

  describe("all adapters referenced in X_HARNESS.md exist", () => {
    const xHarnessPath = path.join(repoRoot, "X_HARNESS.md");
    const xHarnessContent = fs.readFileSync(xHarnessPath, "utf-8");

    for (const adapter of adapters) {
      it(`${adapter.name} is referenced in X_HARNESS.md`, () => {
        expect(xHarnessContent).toContain(adapter.name);
      });
    }
  });

  describe("adapter content contract", () => {
    // Readme files to scan for content contract violations
    const readmeFiles: { adapter: string; path: string }[] = [
      { adapter: "generic", path: "adapters/generic/README.md" },
      { adapter: "claude-code", path: "adapters/claude-code/README.md" },
      { adapter: "cursor", path: "adapters/cursor/README.md" },
      { adapter: "opencode", path: "adapters/opencode/README.md" },
      { adapter: "antigravity", path: "adapters/antigravity/README.md" },
    ];

    const INVALID_GATE_OUTCOME_PHRASES = [
      "verify_gate.outcome",
      "verify-gate.outcome",
    ];
    const INVALID_ZOD_CLAIMS = [
      "zod dependency",
      "depends on zod",
      "uses zod for",
    ];
    const REQUIRED_ADMISSION_SEMANTICS = [
      "admission.outcome",
      "acceptance_status",
      // Fallback: adapters may use "accepted"/"withheld" as the semantic equivalent
      "accepted",
      "withheld",
    ];
    const READ_ONLY_VERIFIER_PHRASES = [
      "read-only verifier",
      "read only verifier",
      "verifier is read-only",
      "strict read-only",
    ];

    for (const { adapter, path: filePath } of readmeFiles) {
      const fullPath = path.join(repoRoot, filePath);
      if (!fs.existsSync(fullPath)) continue;

      const content = fs.readFileSync(fullPath, "utf-8");
      const lower = content.toLowerCase();

      describe(adapter, () => {
        for (const phrase of INVALID_GATE_OUTCOME_PHRASES) {
          it(`does not use ${phrase} (use admission.outcome instead)`, () => {
            expect(content).not.toContain(phrase);
          });
        }

        for (const claim of INVALID_ZOD_CLAIMS) {
          it(`does not claim ${claim}`, () => {
            expect(lower).not.toContain(claim);
          });
        }

        it("includes admission semantics (admission.outcome, acceptance_status, or accepted/withheld)", () => {
          const hasAdmission = REQUIRED_ADMISSION_SEMANTICS.some((term) =>
            content.includes(term)
          );
          expect(
            hasAdmission,
            `${filePath} should mention admission/acceptance semantics`
          ).toBe(true);
        });

        it("mentions read-only verifier", () => {
          const hasReadOnly = READ_ONLY_VERIFIER_PHRASES.some((phrase) =>
            lower.includes(phrase.toLowerCase())
          );
          expect(
            hasReadOnly,
            `${filePath} should mention read-only verifier`
          ).toBe(true);
        });
      });
    }
  });
});
