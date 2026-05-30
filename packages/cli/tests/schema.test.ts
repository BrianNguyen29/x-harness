import { describe, it, expect } from "vitest";
import * as path from "node:path";
import * as fs from "node:fs";
import * as os from "node:os";
import { fileURLToPath } from "node:url";
import { validate as validateClaim } from "../src/validators/claim.js";
import { validate as validateEvidence } from "../src/validators/evidence.js";
import { validate as validateCompletionCard } from "../src/validators/completionCard.js";
import { validate as validateSubagentReturn } from "../src/validators/subagentReturn.js";
import { compileSchema, loadSchema } from "../src/core/schema.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));

describe("schema validators", () => {
  describe("claim schema", () => {
    it("accepts minimal valid claim", async () => {
      const result = await validateClaim({ fix_status: "fixed" });
      expect(result.valid).toBe(true);
    });

    it("accepts claim with all optional fields", async () => {
      const result = await validateClaim({
        id: "C1",
        fix_status: "fixed",
        summary: "Fixed the bug",
        evidence: ["file.ts"],
      });
      expect(result.valid).toBe(true);
    });

    it("rejects invalid fix_status", async () => {
      const result = await validateClaim({ fix_status: "done" });
      expect(result.valid).toBe(false);
    });

    it("rejects missing fix_status", async () => {
      const result = await validateClaim({});
      expect(result.valid).toBe(false);
    });
  });

  describe("evidence schema", () => {
    it("accepts minimal valid evidence", async () => {
      const result = await validateEvidence({
        files_changed: ["src/index.ts"],
      });
      expect(result.valid).toBe(true);
    });

    it("accepts evidence with all optional fields", async () => {
      const result = await validateEvidence({
        id: "E1",
        files_changed: ["src/index.ts"],
        verification_artifacts: [],
        untested_regions: [],
        remaining_risks: [],
        rollback_policy: [],
        execution_controls: [],
      });
      expect(result.valid).toBe(true);
    });

    it("rejects evidence without files_changed", async () => {
      const result = await validateEvidence({});
      expect(result.valid).toBe(false);
    });

    it("rejects files_changed with empty array", async () => {
      const result = await validateEvidence({ files_changed: [] });
      expect(result.valid).toBe(false);
    });

    it("rejects files_changed with non-string items", async () => {
      const result = await validateEvidence({ files_changed: [123] });
      expect(result.valid).toBe(false);
    });
  });

  describe("subagent-return schema", () => {
    it("accepts minimal valid subagent return", async () => {
      const result = await validateSubagentReturn({
        result: { summary: "done", fix_status: "fixed", key_findings: [] },
        evidence: {},
        verification: { status: "passed" },
        confidence: "HIGH",
        handoff: { next_action: "none", owner: "alice" },
      });
      expect(result.valid).toBe(true);
    });

    it("accepts subagent return with timeout status", async () => {
      const result = await validateSubagentReturn({
        result: { summary: "timeout", fix_status: "partial", key_findings: [] },
        evidence: {},
        verification: { status: "timeout" },
        confidence: "MED",
        handoff: { next_action: "retry", owner: "alice" },
      });
      expect(result.valid).toBe(true);
    });

    it("accepts subagent return with error status", async () => {
      const result = await validateSubagentReturn({
        result: { summary: "error", fix_status: "not_fixed", key_findings: [] },
        evidence: {},
        verification: { status: "error" },
        confidence: "LOW",
        handoff: { next_action: "investigate", owner: "bob" },
      });
      expect(result.valid).toBe(true);
    });

    it("rejects invalid verification status", async () => {
      const result = await validateSubagentReturn({
        result: { summary: "done", fix_status: "fixed", key_findings: [] },
        evidence: {},
        verification: { status: "unknown" },
        confidence: "HIGH",
        handoff: { next_action: "none", owner: "alice" },
      });
      expect(result.valid).toBe(false);
    });

    it("rejects invalid confidence", async () => {
      const result = await validateSubagentReturn({
        result: { summary: "done", fix_status: "fixed", key_findings: [] },
        evidence: {},
        verification: { status: "passed" },
        confidence: "MEDIUM",
        handoff: { next_action: "none", owner: "alice" },
      });
      expect(result.valid).toBe(false);
    });
  });

  describe("completion-card schema", () => {
    const minimalCard = {
      schema_version: "1",
      task_id: "T1",
      tier: "light",
      owner: "alice",
      accountable: "bob",
      claim: {
        fix_status: "fixed",
        summary: "done",
        evidence: ["e1"],
      },
      verification: {
        status: "passed",
        checks: [],
      },
      admission: {
        outcome: "success",
      },
      acceptance_status: "accepted",
      handoff: {
        next_action: "none",
        owner: "alice",
      },
    };

    it("accepts minimal valid completion card", async () => {
      const result = await validateCompletionCard(minimalCard);
      expect(result.valid).toBe(true);
    });

    it("accepts card with all optional fields", async () => {
      const card = {
        ...minimalCard,
        id: "card-1",
        state: {
          read_set: ["a.ts"],
          write_set: ["b.ts"],
          assumptions: [],
          conflict_policy: {},
        },
        evidence: {
          files_changed: ["a.ts"],
          verification_artifacts: [],
          untested_regions: [],
          remaining_risks: [],
          rollback_policy: [],
          execution_controls: [],
        },
        governance: {
          risk_class: "low",
          requires_human_approval: false,
          approval_required_for: [],
          approval_status: "not_required",
          approver: undefined,
        },
        intake: {
          classification: "normal",
          mapped_tier: "standard",
          rationale: "Routine implementation",
          signals: ["routine_implementation"],
          negative_signals_considered: ["auth", "token"],
          auto_escalated: false,
        },
        context_acknowledged: true,
      };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(true);
    });

    it("rejects card with invalid tier", async () => {
      const card = { ...minimalCard, tier: "small" };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(false);
    });

    it("rejects card with invalid intake mapped_tier", async () => {
      const card = {
        ...minimalCard,
        intake: {
          classification: "high_risk",
          mapped_tier: "large",
          rationale: "Invalid runtime tier",
        },
      };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(false);
    });

    it("rejects card with invalid acceptance_status", async () => {
      const card = { ...minimalCard, acceptance_status: "pending" };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(false);
    });

    it("rejects accepted-without-success admission", async () => {
      const card = {
        ...minimalCard,
        admission: { outcome: "failed" },
        acceptance_status: "accepted",
      };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(false);
    });

    it("rejects success admission with withheld acceptance", async () => {
      const card = {
        ...minimalCard,
        admission: { outcome: "success" },
        acceptance_status: "withheld",
      };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(false);
    });

    it("rejects success admission without passed verification", async () => {
      const card = {
        ...minimalCard,
        verification: { status: "blocked", checks: [] },
        admission: { outcome: "success" },
        acceptance_status: "accepted",
      };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(false);
    });

    it("rejects empty claim evidence", async () => {
      const card = {
        ...minimalCard,
        claim: { ...minimalCard.claim, evidence: [] },
      };
      const result = await validateCompletionCard(card);
      expect(result.valid).toBe(false);
    });
  });

  describe("cross-schema sync", () => {
    // Schemas that should be byte-identical between root (published contract) and runtime copies
    const syncedSchemas = [
      "adapter-matrix.schema.json",
      "approval-receipt.schema.json",
      "attribution.schema.json",
      "agent-profile.schema.json",
      "approval-risk.schema.json",
      "benchmark-report.schema.json",
      "classifier.schema.json",
      "completion-card.schema.json",
      "components-registry.schema.json",
      "contract-oracle.schema.json",
      "cost-budget.schema.json",
      "evidence-index.schema.json",
      "evolution-constitution.schema.json",
      "episode-manifest.schema.json",
      "frozen-manifest.schema.json",
      "federation-pattern.schema.json",
      "permissions.schema.json",
      "release-evidence.schema.json",
      "report.schema.json",
      "scanner.schema.json",
      "subagent-return.schema.json",
      "verify-event.schema.json",
      "pgv-advice.schema.json",
      "claim.schema.json",
      "evidence.schema.json",
      "packet.schema.json",
      "intervention.schema.json",
      "withheld-reason.schema.json",
    ];

    // Helper to compare two schema files and return whether they match
    function compareSchemaFiles(
      rootPath: string,
      runtimePath: string
    ): { match: boolean; rootContent: string; runtimeContent: string } {
      const rootContent = fs.readFileSync(rootPath, "utf-8");
      const runtimeContent = fs.readFileSync(runtimePath, "utf-8");
      return {
        match: rootContent === runtimeContent,
        rootContent,
        runtimeContent,
      };
    }

    for (const schemaFile of syncedSchemas) {
      it(`${schemaFile} runtime copy matches root contract`, () => {
        const rootPath = path.join(repoRoot, "schemas", schemaFile);
        const runtimePath = path.join(
          repoRoot,
          "packages",
          "cli",
          "schemas",
          schemaFile
        );
        if (!fs.existsSync(rootPath)) {
          // Schema only exists in runtime (e.g. runtime-only validator)
          return;
        }
        if (!fs.existsSync(runtimePath)) {
          expect.fail(
            `${schemaFile} exists in root but not in packages/cli/schemas`
          );
          return;
        }
        const { match, rootContent, runtimeContent } = compareSchemaFiles(
          rootPath,
          runtimePath
        );
        expect(
          match,
          `${schemaFile} should be byte-identical in root and runtime. ` +
            `Diff: root has ${rootContent.length} chars, runtime has ${runtimeContent.length} chars`
        ).toBe(true);
      });
    }

    it("schema sync helper detects drift when schemas differ", () => {
      // This test proves the comparison helper can detect drift
      const testRoot = path.join(os.tmpdir(), "schema-test-root-" + Date.now());
      const testRuntime = path.join(
        os.tmpdir(),
        "schema-test-runtime-" + Date.now()
      );
      fs.mkdirSync(testRoot, { recursive: true });
      fs.mkdirSync(testRuntime, { recursive: true });

      try {
        const schemaName = "test-schema.json";
        const rootPath = path.join(testRoot, schemaName);
        const runtimePath = path.join(testRuntime, schemaName);

        // Write identical content - should match
        fs.writeFileSync(rootPath, '{"type": "object"}');
        fs.writeFileSync(runtimePath, '{"type": "object"}');
        const identical = compareSchemaFiles(rootPath, runtimePath);
        expect(identical.match).toBe(true);

        // Write different content - should NOT match (proves drift detection works)
        fs.writeFileSync(runtimePath, '{"type": "array"}');
        const different = compareSchemaFiles(rootPath, runtimePath);
        expect(different.match).toBe(false);
      } finally {
        fs.rmSync(testRoot, { recursive: true, force: true });
        fs.rmSync(testRuntime, { recursive: true, force: true });
      }
    });
  });

  describe("benchmark report schema", () => {
    it("accepts minimal benchmark report metrics", async () => {
      const schema = await loadSchema("benchmark-report");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: true,
        generated_at: "2026-05-24T00:00:00.000Z",
        iterations: 1,
        timeout_ms: 120000,
        filter: "admission",
        results: [],
        integration: null,
        metrics: {
          false_accept_count: 0,
          false_reject_count: 0,
          expected_pass_count: 0,
          expected_block_count: 0,
          schema_validation_pass_rate: null,
          policy_validation_pass_rate: null,
          episode_packaging_success_rate: null,
          mutation_guard_detection_rate: null,
          permission_violation_detection_rate: null,
          adversarial_false_accept_count: 0,
          adversarial_block_rate: null,
          runtime_ms: 0,
        },
      });
      expect(validate.errors ?? []).toEqual([]);
      expect(valid).toBe(true);
    });
  });

  describe("report schema", () => {
    it("accepts a metrics report", async () => {
      const schema = await loadSchema("report");
      const validate = compileSchema(schema);
      const valid = validate({
        card_id: null,
        task_id: "TASK-1",
        tier: "light",
        metrics: {
          verification_strength: {
            command_evidence_count: 1,
            oracle_kinds: ["unit_test"],
            untested_regions_count: 0,
            remaining_risks_count: 0,
          },
          state_consistency: {
            owner_present: true,
            accountable_present: true,
            files_changed_present: true,
            admission_mapping_valid: true,
          },
          recovery_ability: {
            blocked_has_next_action: false,
            blocked_has_owner: false,
            recovery_route_present: false,
          },
          replayability: {
            completion_card_present: true,
            input_card_hash_present: true,
            policy_hash_present: true,
          },
          cost: {
            default_context_class: "standard",
            verify_runtime_ms: 1,
          },
          verify_event_success_rate: {
            numerator: 1,
            denominator: 1,
            unit: "verify_event",
            not_task_level: true,
          },
          task_completion_coverage: {
            status: "not_computable",
            reason: "missing_aligned_task_denominator",
          },
          withheld_rate: {
            numerator: 0,
            denominator: 1,
            unit: "verify_event",
            not_task_level: true,
          },
        },
        admission: {
          outcome: "success",
          acceptance_status: "accepted",
          errors: [],
          notes: [],
        },
        verify_event_accounting: {
          cards_analyzed: 1,
          note: "Single-card analysis.",
        },
        task_lifecycle_accounting: {
          admitted: 1,
          withheld: 0,
          note: "Lifecycle state reflects only the analyzed completion card.",
        },
        admission_accounting: {
          accepted: 1,
          total_analyzed: 1,
          note: "Admission requires success.",
        },
        withheld_accounting: {
          failed: 0,
          blocked: 0,
          skipped: 0,
          timeout: 0,
          error: 0,
          note: "None.",
        },
        unknown_or_unlinked_events: {
          count: 0,
          note: "None.",
        },
        denominator_warning: "Do not infer task-level success.",
      });
      expect(validate.errors ?? []).toEqual([]);
      expect(valid).toBe(true);
    });

    it("accepts a trace report", async () => {
      const schema = await loadSchema("report");
      const validate = compileSchema(schema);
      const valid = validate({
        total_events: 1,
        accepted: 1,
        withheld: 0,
        by_outcome: { success: 1 },
        verify_event_accounting: {
          total_trace_events: 1,
          note: "Counts are based only on traced verify events.",
        },
        task_lifecycle_accounting: {
          admitted: 1,
          withheld: 0,
          note: "Lifecycle accounting covers only events present in the trace log.",
        },
        admission_accounting: {
          accepted: 1,
          total_trace_events: 1,
          note: "Admission requires success.",
        },
        withheld_accounting: {
          failed: 0,
          blocked: 0,
          skipped: 0,
          timeout: 0,
          error: 0,
          note: "None.",
        },
        unknown_or_unlinked_events: {
          count: 0,
          note: "None.",
        },
        latest: null,
        verify_event_success_rate: {
          numerator: 1,
          denominator: 1,
          unit: "verify_event",
          not_task_level: true,
        },
        task_completion_coverage: {
          status: "not_computable",
          reason: "missing_aligned_task_denominator",
        },
        withheld_rate: {
          numerator: 0,
          denominator: 1,
          unit: "verify_event",
          not_task_level: true,
        },
      });
      expect(validate.errors ?? []).toEqual([]);
      expect(valid).toBe(true);
    });
  });

  describe("withheld-reason schema", () => {
    it("accepts minimal valid withheld-reason", async () => {
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const valid = validate({
        class: "unresolved_blocker",
        blocking_predicate: "missing_required_field",
        stage: "admission",
        recoverability: "manual",
        owner: "alice",
        next_action: "add_required_field",
      });
      expect(validate.errors ?? []).toEqual([]);
      expect(valid).toBe(true);
    });

    it("accepts withheld-reason with all optional fields", async () => {
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const valid = validate({
        class: "evidence_floor_missing",
        blocking_predicate: "files_changed_empty",
        stage: "evidence",
        recoverability: "automatic",
        schema_recoverability: "manual",
        owner: "bob",
        next_action: "add_evidence",
        missing_field: "files_changed",
        policy_path: "policies/admission.yaml",
        trace_event_id: "evt_123",
      });
      expect(validate.errors ?? []).toEqual([]);
      expect(valid).toBe(true);
    });

    it("rejects missing required field", async () => {
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const valid = validate({
        class: "unresolved_blocker",
        blocking_predicate: "missing_required_field",
        stage: "admission",
        recoverability: "manual",
        owner: "alice",
        // missing next_action
      });
      expect(valid).toBe(false);
    });

    it("rejects invalid enum value for class", async () => {
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const valid = validate({
        class: "invalid_class",
        blocking_predicate: "test",
        stage: "admission",
        recoverability: "manual",
        owner: "alice",
        next_action: "fix",
      });
      expect(valid).toBe(false);
    });

    it("rejects invalid enum value for stage", async () => {
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const valid = validate({
        class: "unresolved_blocker",
        blocking_predicate: "test",
        stage: "invalid_stage",
        recoverability: "manual",
        owner: "alice",
        next_action: "fix",
      });
      expect(valid).toBe(false);
    });

    it("rejects invalid enum value for recoverability", async () => {
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const valid = validate({
        class: "unresolved_blocker",
        blocking_predicate: "test",
        stage: "admission",
        recoverability: "invalid_recoverability",
        owner: "alice",
        next_action: "fix",
      });
      expect(valid).toBe(false);
    });

    it("rejects additional properties", async () => {
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const valid = validate({
        class: "unresolved_blocker",
        blocking_predicate: "test",
        stage: "admission",
        recoverability: "manual",
        owner: "alice",
        next_action: "fix",
        extra_field: "not_allowed",
      });
      expect(valid).toBe(false);
    });

    it("rejects empty blocking_predicate", async () => {
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const valid = validate({
        class: "unresolved_blocker",
        blocking_predicate: "",
        stage: "admission",
        recoverability: "manual",
        owner: "alice",
        next_action: "fix",
      });
      expect(valid).toBe(false);
    });

    it("accepts schema_recoverability with valid enum value", async () => {
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const valid = validate({
        class: "unresolved_blocker",
        blocking_predicate: "test",
        stage: "admission",
        recoverability: "manual",
        schema_recoverability: "automatic",
        owner: "alice",
        next_action: "fix",
      });
      expect(validate.errors ?? []).toEqual([]);
      expect(valid).toBe(true);
    });

    it("rejects schema_recoverability with invalid enum value", async () => {
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const valid = validate({
        class: "unresolved_blocker",
        blocking_predicate: "test",
        stage: "admission",
        recoverability: "manual",
        schema_recoverability: "invalid_value",
        owner: "alice",
        next_action: "fix",
      });
      expect(valid).toBe(false);
    });

    it("accepts strict-schema-only withheld-reason canonical target (no legacy fields)", async () => {
      // This is the canonical strict-schema target: only schema fields, no legacy fields.
      // Runtime superset includes legacy failure_class/failure_stage which are rejected here.
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const fixture = JSON.parse(fs.readFileSync(path.join(__dirname, "fixtures", "withheld-reason-strict.json"), "utf-8"));
      const valid = validate(fixture);
      expect(validate.errors ?? []).toEqual([]);
      expect(valid).toBe(true);
    });

    it("rejects runtime compatibility superset with legacy fields", async () => {
      // Runtime Go output includes legacy failure_class/failure_stage which are not in the strict schema.
      const schema = await loadSchema("withheld-reason");
      const validate = compileSchema(schema);
      const valid = validate({
        class: "evidence_floor_missing",
        blocking_predicate: "files_changed_empty",
        stage: "evidence",
        recoverability: "retry_with_fixes",
        schema_recoverability: "manual",
        owner: "bob",
        next_action: "add_evidence",
        failure_class: "missing_files_changed",   // legacy field, not in schema
        failure_stage: "evidence",                // legacy field, not in schema
      });
      expect(valid).toBe(false);
    });
  });

  describe("contract-oracle schema", () => {
    it("accepts minimal valid contract-oracle result", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: true,
        policy: "policy.yaml",
        files_scanned: 0,
        violations: [],
      });
      expect(validate.errors ?? []).toEqual([]);
      expect(valid).toBe(true);
    });

    it("accepts result with violations", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: false,
        policy: "policy.yaml",
        files_scanned: 5,
        violations: [
          {
            rule_id: "no-debug",
            file: "src/main.go",
            line: 42,
            snippet: "fmt.Println(\"debug\")",
            message: "Debug print statements are not allowed",
          },
        ],
      });
      expect(validate.errors ?? []).toEqual([]);
      expect(valid).toBe(true);
    });

    it("rejects missing ok field", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        policy: "policy.yaml",
        files_scanned: 0,
        violations: [],
      });
      expect(valid).toBe(false);
    });

    it("rejects missing policy field", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: true,
        files_scanned: 0,
        violations: [],
      });
      expect(valid).toBe(false);
    });

    it("rejects missing files_scanned field", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: true,
        policy: "policy.yaml",
        violations: [],
      });
      expect(valid).toBe(false);
    });

    it("rejects missing violations field", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: true,
        policy: "policy.yaml",
        files_scanned: 0,
      });
      expect(valid).toBe(false);
    });

    it("rejects violation with missing rule_id", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: false,
        policy: "policy.yaml",
        files_scanned: 1,
        violations: [
          {
            file: "src/main.go",
            line: 42,
            snippet: "debug",
            message: "No rule_id",
          },
        ],
      });
      expect(valid).toBe(false);
    });

    it("rejects violation with missing file", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: false,
        policy: "policy.yaml",
        files_scanned: 1,
        violations: [
          {
            rule_id: "no-debug",
            line: 42,
            snippet: "debug",
            message: "No file",
          },
        ],
      });
      expect(valid).toBe(false);
    });

    it("rejects violation with missing line", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: false,
        policy: "policy.yaml",
        files_scanned: 1,
        violations: [
          {
            rule_id: "no-debug",
            file: "src/main.go",
            snippet: "debug",
            message: "No line",
          },
        ],
      });
      expect(valid).toBe(false);
    });

    it("rejects violation with line less than 1", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: false,
        policy: "policy.yaml",
        files_scanned: 1,
        violations: [
          {
            rule_id: "no-debug",
            file: "src/main.go",
            line: 0,
            snippet: "debug",
            message: "Line must be >= 1",
          },
        ],
      });
      expect(valid).toBe(false);
    });

    it("rejects violation with missing snippet", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: false,
        policy: "policy.yaml",
        files_scanned: 1,
        violations: [
          {
            rule_id: "no-debug",
            file: "src/main.go",
            line: 42,
            message: "No snippet",
          },
        ],
      });
      expect(valid).toBe(false);
    });

    it("rejects violation with missing message", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: false,
        policy: "policy.yaml",
        files_scanned: 1,
        violations: [
          {
            rule_id: "no-debug",
            file: "src/main.go",
            line: 42,
            snippet: "debug",
          },
        ],
      });
      expect(valid).toBe(false);
    });

    it("rejects additional properties on result", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: true,
        policy: "policy.yaml",
        files_scanned: 0,
        violations: [],
        extra_field: "not allowed",
      });
      expect(valid).toBe(false);
    });

    it("rejects additional properties on violation", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: false,
        policy: "policy.yaml",
        files_scanned: 1,
        violations: [
          {
            rule_id: "no-debug",
            file: "src/main.go",
            line: 42,
            snippet: "debug",
            message: "msg",
            extra: "not allowed",
          },
        ],
      });
      expect(valid).toBe(false);
    });

    it("accepts files_scanned as zero", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: true,
        policy: "policy.yaml",
        files_scanned: 0,
        violations: [],
      });
      expect(validate.errors ?? []).toEqual([]);
      expect(valid).toBe(true);
    });

    it("accepts multiple violations", async () => {
      const schema = await loadSchema("contract-oracle");
      const validate = compileSchema(schema);
      const valid = validate({
        ok: false,
        policy: "policy.yaml",
        files_scanned: 2,
        violations: [
          { rule_id: "no-debug", file: "a.go", line: 1, snippet: "debug1", message: "msg1" },
          { rule_id: "no-debug", file: "b.go", line: 5, snippet: "debug2", message: "msg2" },
          { rule_id: "no-print", file: "a.go", line: 10, snippet: "print", message: "msg3" },
        ],
      });
      expect(validate.errors ?? []).toEqual([]);
      expect(valid).toBe(true);
    });
  });
});
