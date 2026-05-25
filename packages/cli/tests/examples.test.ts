import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("examples command", () => {
  it("verify subcommand returns JSON with all examples", async () => {
    const { stdout, exitCode } = await execaNode([
      "examples",
      "verify",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.ok).toBe(true);
    expect(output.total).toBe(12);
    expect(output.passed).toBe(12);
    expect(output.failed).toBe(0);
    expect(output.results).toHaveLength(12);

    const names = output.results.map((r: { name: string }) => r.name);
    expect(names).toContain("success-light");
    expect(names).toContain("blocked-missing-evidence");
    expect(names).toContain("failed-invalid-status");
    expect(names).toContain("withheld-partial-fix");
    expect(names).toContain("multi-agent-success");
    expect(names).toContain("success-standard-scoped-evidence");
    expect(names).toContain("blocked-missing-evidence-scope");
    expect(names).toContain("deep-approval-required");
    expect(names).toContain("failed-typecheck-recovery-route");
    expect(names).toContain("blocked-missing-done-checklist");
    expect(names).toContain("blocked-weak-prediction");
    expect(names).toContain("blocked-tier-downgrade");
  });

  it("verify subcommand prints human-readable summary", async () => {
    const { stdout, exitCode } = await execaNode(["examples", "verify"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Golden examples: 12 total");
    expect(stdout).toContain("success-light");
    expect(stdout).toContain("blocked-missing-evidence");
    expect(stdout).toContain("failed-invalid-status");
    expect(stdout).toContain("withheld-partial-fix");
    expect(stdout).toContain("multi-agent-success");
    expect(stdout).toContain("success-standard-scoped-evidence");
    expect(stdout).toContain("blocked-missing-evidence-scope");
    expect(stdout).toContain("deep-approval-required");
    expect(stdout).toContain("failed-typecheck-recovery-route");
    expect(stdout).toContain("blocked-missing-done-checklist");
    expect(stdout).toContain("blocked-weak-prediction");
    expect(stdout).toContain("blocked-tier-downgrade");
    expect(stdout).toContain("All golden examples passed.");
  });

  it("each golden example has expected outcome", async () => {
    const { stdout, exitCode } = await execaNode([
      "examples",
      "verify",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);

    const successLight = output.results.find(
      (r: { name: string }) => r.name === "success-light"
    );
    expect(successLight.outcome).toBe("success");
    expect(successLight.acceptance_status).toBe("accepted");

    const blocked = output.results.find(
      (r: { name: string }) => r.name === "blocked-missing-evidence"
    );
    expect(blocked.outcome).toBe("failed");
    expect(blocked.acceptance_status).toBe("withheld");

    const tierDowngrade = output.results.find(
      (r: { name: string }) => r.name === "blocked-tier-downgrade"
    );
    expect(tierDowngrade.outcome).toBe("failed");
    expect(tierDowngrade.acceptance_status).toBe("withheld");

    const failed = output.results.find(
      (r: { name: string }) => r.name === "failed-invalid-status"
    );
    expect(failed.outcome).toBe("failed");
    expect(failed.acceptance_status).toBe("withheld");

    const withheld = output.results.find(
      (r: { name: string }) => r.name === "withheld-partial-fix"
    );
    expect(withheld.outcome).toBe("failed");
    expect(withheld.acceptance_status).toBe("withheld");

    const multi = output.results.find(
      (r: { name: string }) => r.name === "multi-agent-success"
    );
    expect(multi.outcome).toBe("success");
    expect(multi.acceptance_status).toBe("accepted");

    const scoped = output.results.find(
      (r: { name: string }) => r.name === "success-standard-scoped-evidence"
    );
    expect(scoped.outcome).toBe("success");
    expect(scoped.acceptance_status).toBe("accepted");

    const blockedScope = output.results.find(
      (r: { name: string }) => r.name === "blocked-missing-evidence-scope"
    );
    expect(blockedScope.outcome).toBe("failed");
    expect(blockedScope.acceptance_status).toBe("withheld");

    const approval = output.results.find(
      (r: { name: string }) => r.name === "deep-approval-required"
    );
    expect(approval.outcome).toBe("failed");
    expect(approval.acceptance_status).toBe("withheld");

    const recovery = output.results.find(
      (r: { name: string }) => r.name === "failed-typecheck-recovery-route"
    );
    expect(recovery.outcome).toBe("failed");
    expect(recovery.acceptance_status).toBe("withheld");

    const missingChecklist = output.results.find(
      (r: { name: string }) => r.name === "blocked-missing-done-checklist"
    );
    expect(missingChecklist.outcome).toBe("failed");
    expect(missingChecklist.acceptance_status).toBe("withheld");

    const weakPrediction = output.results.find(
      (r: { name: string }) => r.name === "blocked-weak-prediction"
    );
    expect(weakPrediction.outcome).toBe("failed");
    expect(weakPrediction.acceptance_status).toBe("withheld");
  });
});
