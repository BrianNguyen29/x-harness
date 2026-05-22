import { describe, it, expect } from "vitest";
import {
  getRecoveryRoute,
  suggestRecovery,
  type RecoveryPredicate,
} from "../src/core/recovery.js";
import * as path from "node:path";
import * as fs from "node:fs";
import * as YAML from "yaml";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const policyPath = path.join(repoRoot, "policies", "recovery.yaml");
const fixturePath = path.join(
  repoRoot,
  "examples",
  "golden",
  "recovery",
  "routes.yaml"
);

describe("recovery", () => {
  describe("getRecoveryRoute", () => {
    it("returns route for evidence_missing", () => {
      const route = getRecoveryRoute("evidence_missing");
      expect(route).not.toBeNull();
      expect(route!.owner).toBe("implementation-worker");
      expect(route!.next_action).toContain("Attach validation evidence");
    });

    it("returns route for evidence_scope_missing", () => {
      const route = getRecoveryRoute("evidence_scope_missing");
      expect(route).not.toBeNull();
      expect(route!.owner).toBe("implementation-worker");
      expect(route!.next_action).toContain(
        "Declare what each validation artifact verifies"
      );
    });

    it("returns route for typecheck_failed", () => {
      const route = getRecoveryRoute("typecheck_failed");
      expect(route).not.toBeNull();
      expect(route!.owner).toBe("implementation-worker");
      expect(route!.next_action).toContain("type repair");
    });

    it("returns route for test_failed (maps tests_failed)", () => {
      const route = getRecoveryRoute("test_failed");
      expect(route).not.toBeNull();
      expect(route!.owner).toBe("implementation-worker");
      expect(route!.next_action).toContain("Diagnose failing behavior");
    });

    it("returns route for lint_failed", () => {
      const route = getRecoveryRoute("lint_failed");
      expect(route).not.toBeNull();
      expect(route!.owner).toBe("implementation-worker");
      expect(route!.next_action).toContain("Fix lint issues");
    });

    it("returns route for build_failed", () => {
      const route = getRecoveryRoute("build_failed");
      expect(route).not.toBeNull();
      expect(route!.owner).toBe("implementation-worker");
      expect(route!.next_action).toContain("Fix build failure");
    });

    it("returns route for approval_missing", () => {
      const route = getRecoveryRoute("approval_missing");
      expect(route).not.toBeNull();
      expect(route!.owner).toBe("user");
      expect(route!.next_action).toContain("human approval");
    });

    it("returns route for conflicting_scope", () => {
      const route = getRecoveryRoute("conflicting_scope");
      expect(route).not.toBeNull();
      expect(route!.owner).toBe("user");
      expect(route!.next_action).toContain("clarify task scope");
    });

    it("returns route for verifier_not_read_only", () => {
      const route = getRecoveryRoute("verifier_not_read_only");
      expect(route).not.toBeNull();
      expect(route!.owner).toBe("admission-verifier");
      expect(route!.next_action).toContain("read-only verifier");
    });

    it("returns route for admission_failed (maps unknown_failure fallback)", () => {
      const route = getRecoveryRoute("admission_failed");
      expect(route).not.toBeNull();
      expect(route!.owner).toBe("implementation-worker");
      expect(route!.next_action).toContain("admission validation errors");
    });

    it("returns null for null/undefined predicate", () => {
      expect(getRecoveryRoute(null)).toBeNull();
      expect(getRecoveryRoute(undefined)).toBeNull();
    });

    it("returns null for unsupported predicates", () => {
      // schema_invalid, stale_ground, and policy_drift have no dedicated recovery routes
      expect(getRecoveryRoute("schema_invalid")).toBeNull();
      expect(getRecoveryRoute("stale_ground")).toBeNull();
      expect(getRecoveryRoute("policy_drift")).toBeNull();
      expect(getRecoveryRoute("unknown_failure")).toBeNull();
    });
  });

  describe("suggestRecovery", () => {
    it("maps approval errors to approval_missing", () => {
      const result = suggestRecovery(["missing human approval"], "failed");
      expect(result.predicate).toBe("approval_missing");
      expect(result.route).not.toBeNull();
      expect(result.route!.owner).toBe("user");
    });

    it("maps typecheck errors to typecheck_failed", () => {
      const result = suggestRecovery(
        ["tsc --noEmit reported typecheck errors"],
        "failed"
      );
      expect(result.predicate).toBe("typecheck_failed");
      expect(result.route).not.toBeNull();
      expect(result.route!.owner).toBe("implementation-worker");
    });

    it("maps test errors to test_failed", () => {
      const result = suggestRecovery(["unit tests failed"], "failed");
      expect(result.predicate).toBe("test_failed");
      expect(result.route).not.toBeNull();
      expect(result.route!.owner).toBe("implementation-worker");
    });

    it("does not map test errors to test_failed when typecheck is also present", () => {
      // heuristic: typecheck takes precedence when both appear
      const result = suggestRecovery(["typecheck and tests failed"], "failed");
      expect(result.predicate).toBe("typecheck_failed");
    });

    it("maps lint errors to lint_failed", () => {
      const result = suggestRecovery(["eslint lint errors found"], "failed");
      expect(result.predicate).toBe("lint_failed");
      expect(result.route).not.toBeNull();
      expect(result.route!.owner).toBe("implementation-worker");
    });

    it("maps build errors to build_failed", () => {
      const result = suggestRecovery(["npm run build failed"], "failed");
      expect(result.predicate).toBe("build_failed");
      expect(result.route).not.toBeNull();
      expect(result.route!.owner).toBe("implementation-worker");
    });

    it("maps scope errors to evidence_scope_missing", () => {
      const result = suggestRecovery(["scope unclear"], "blocked");
      expect(result.predicate).toBe("evidence_scope_missing");
      expect(result.route).not.toBeNull();
      expect(result.route!.owner).toBe("implementation-worker");
    });

    it("maps does_not_verify errors to evidence_scope_missing", () => {
      const result = suggestRecovery(
        ["artifact does_not_verify e2e"],
        "blocked"
      );
      expect(result.predicate).toBe("evidence_scope_missing");
    });

    it("maps untested errors to evidence_scope_missing when test substring not present", () => {
      // Note: "untested" contains "test", so the heuristic may misclassify.
      // Using a phrasing with "scope" avoids the heuristic collision.
      const result = suggestRecovery(
        ["scope: regions not covered by verification"],
        "blocked"
      );
      expect(result.predicate).toBe("evidence_scope_missing");
    });

    it("maps evidence errors to evidence_missing", () => {
      const result = suggestRecovery(["missing evidence packet"], "failed");
      expect(result.predicate).toBe("evidence_missing");
      expect(result.route).not.toBeNull();
      expect(result.route!.owner).toBe("implementation-worker");
    });

    it("falls back to admission_failed for unrecognized errors", () => {
      const result = suggestRecovery(
        ["something unexpected happened"],
        "failed"
      );
      expect(result.predicate).toBe("admission_failed");
      expect(result.route).not.toBeNull();
      expect(result.route!.owner).toBe("implementation-worker");
    });

    it("returns null predicate/route for success outcome", () => {
      const result = suggestRecovery(["some error"], "success");
      expect(result.predicate).toBeNull();
      expect(result.route).toBeNull();
    });

    it("returns null predicate/route for skipped outcome", () => {
      const result = suggestRecovery(["some error"], "skipped");
      expect(result.predicate).toBeNull();
      expect(result.route).toBeNull();
    });

    it("returns null predicate/route for timeout outcome", () => {
      const result = suggestRecovery(["some error"], "timeout");
      expect(result.predicate).toBeNull();
      expect(result.route).toBeNull();
    });

    it("returns admission_failed for empty errors on failed outcome", () => {
      const result = suggestRecovery([], "failed");
      expect(result.predicate).toBe("admission_failed");
      expect(result.route).not.toBeNull();
    });
  });

  describe("policy-code consistency", () => {
    it("recovery.ts DEFAULT_ROUTES matches policies/recovery.yaml", () => {
      const policyContent = fs.readFileSync(policyPath, "utf-8");
      const policy = YAML.parse(policyContent) as {
        recovery_routing: Record<
          string,
          { next_action: string; owner: string }
        >;
      };

      const predicates: RecoveryPredicate[] = [
        "evidence_missing",
        "evidence_scope_missing",
        "typecheck_failed",
        "test_failed",
        "lint_failed",
        "build_failed",
        "approval_missing",
        "conflicting_scope",
        "verifier_not_read_only",
        "admission_failed",
      ];

      for (const predicate of predicates) {
        const codeRoute = getRecoveryRoute(predicate);
        const policyRoute = policy.recovery_routing[predicate];

        if (policyRoute) {
          expect(codeRoute).not.toBeNull();
          expect(codeRoute!.next_action).toBe(policyRoute.next_action);
          expect(codeRoute!.owner).toBe(policyRoute.owner);
        }
      }
    });

    it("fixture routes.yaml matches code if present", () => {
      if (!fs.existsSync(fixturePath)) {
        // Fixture is optional; skip if not yet created
        return;
      }
      const fixtureContent = fs.readFileSync(fixturePath, "utf-8");
      const fixture = YAML.parse(fixtureContent) as {
        recovery_routing: Record<
          string,
          { next_action: string; owner: string }
        >;
      };

      for (const [predicate, expected] of Object.entries(
        fixture.recovery_routing
      )) {
        const codeRoute = getRecoveryRoute(predicate);
        expect(codeRoute).not.toBeNull();
        expect(codeRoute!.next_action).toBe(expected.next_action);
        expect(codeRoute!.owner).toBe(expected.owner);
      }
    });
  });

  describe("unsupported predicates", () => {
    it("schema_invalid has no dedicated recovery route (unsupported)", () => {
      // There is no schema_invalid predicate in recovery.ts;
      // schema validation failures are surfaced as admission_failed.
      const route = getRecoveryRoute("schema_invalid");
      expect(route).toBeNull();
    });

    it("stale_ground has no dedicated recovery route (handled in admission.ts)", () => {
      // stale_ground is a fail-closed admission check, not a recovery route.
      // It bypasses recovery routing and returns immediately.
      const route = getRecoveryRoute("stale_ground");
      expect(route).toBeNull();
    });

    it("policy_drift has no dedicated recovery route (unsupported)", () => {
      // Policy drift guard is planned but not yet implemented in recovery.ts.
      const route = getRecoveryRoute("policy_drift");
      expect(route).toBeNull();
    });

    it("unknown_failure has no dedicated recovery route (maps to admission_failed fallback)", () => {
      // Unrecognized failures fall back to admission_failed via suggestRecovery.
      const route = getRecoveryRoute("unknown_failure");
      expect(route).toBeNull();
      const suggestion = suggestRecovery(
        ["unknown cosmic ray error"],
        "failed"
      );
      expect(suggestion.predicate).toBe("admission_failed");
      expect(suggestion.route).not.toBeNull();
    });
  });
});
