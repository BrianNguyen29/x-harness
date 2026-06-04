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
    expect(output.total).toBe(20);
    expect(output.passed).toBe(20);
    expect(output.results).toHaveLength(20);

    const names = output.results.map((r: { name: string }) => r.name);
    expect(names).toContain("regression/success-light");
    expect(names).toContain("regression/blocked-missing-evidence");
    expect(names).toContain("regression/failed-invalid-status");
    expect(names).toContain("capability/withheld-partial-fix");
    expect(names).toContain("capability/multi-agent-success");
    expect(names).toContain("regression/success-standard-scoped-evidence");
    expect(names).toContain("regression/blocked-missing-evidence-scope");
    expect(names).toContain("capability/deep-approval-required");
    expect(names).toContain("capability/failed-typecheck-recovery-route");
    expect(names).toContain("adversarial/blocked-missing-done-checklist");
    expect(names).toContain("adversarial/standard-approval-missing");
    expect(names).toContain("adversarial/standard-approval-present");
    expect(names).toContain("regression/blocked-weak-prediction");
    expect(names).toContain("regression/blocked-tier-downgrade");
    expect(names).toContain("regression/success-context-alignment");
    expect(names).toContain("regression/blocked-missing-context-ref");
    expect(names).toContain("regression/blocked-contract-oracle");
    expect(names).toContain("regression/success-recovered-flow");
    expect(names).toContain("regression/boundary-allow");
    expect(names).toContain("regression/boundary-violation");
  });

  it("verify subcommand prints human-readable summary", async () => {
    const { stdout, exitCode } = await execaNode(["examples", "verify"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Golden examples: 20 total");
    expect(stdout).toContain("regression/success-light");
    expect(stdout).toContain("regression/blocked-missing-evidence");
    expect(stdout).toContain("regression/failed-invalid-status");
    expect(stdout).toContain("capability/withheld-partial-fix");
    expect(stdout).toContain("capability/multi-agent-success");
    expect(stdout).toContain("regression/success-standard-scoped-evidence");
    expect(stdout).toContain("regression/blocked-missing-evidence-scope");
    expect(stdout).toContain("capability/deep-approval-required");
    expect(stdout).toContain("capability/failed-typecheck-recovery-route");
    expect(stdout).toContain("adversarial/blocked-missing-done-checklist");
    expect(stdout).toContain("adversarial/standard-approval-missing");
    expect(stdout).toContain("adversarial/standard-approval-present");
    expect(stdout).toContain("regression/blocked-weak-prediction");
    expect(stdout).toContain("regression/blocked-tier-downgrade");
    expect(stdout).toContain("regression/success-context-alignment");
    expect(stdout).toContain("regression/blocked-missing-context-ref");
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
      (r: { name: string }) => r.name === "regression/success-light"
    );
    expect(successLight.outcome).toBe("success");
    expect(successLight.acceptance_status).toBe("accepted");

    const blocked = output.results.find(
      (r: { name: string }) => r.name === "regression/blocked-missing-evidence"
    );
    expect(blocked.outcome).toBe("failed");
    expect(blocked.acceptance_status).toBe("withheld");

    const tierDowngrade = output.results.find(
      (r: { name: string }) => r.name === "regression/blocked-tier-downgrade"
    );
    expect(tierDowngrade.outcome).toBe("failed");
    expect(tierDowngrade.acceptance_status).toBe("withheld");

    const failed = output.results.find(
      (r: { name: string }) => r.name === "regression/failed-invalid-status"
    );
    expect(failed.outcome).toBe("failed");
    expect(failed.acceptance_status).toBe("withheld");

    const withheld = output.results.find(
      (r: { name: string }) => r.name === "capability/withheld-partial-fix"
    );
    expect(withheld.outcome).toBe("failed");
    expect(withheld.acceptance_status).toBe("withheld");

    const multi = output.results.find(
      (r: { name: string }) => r.name === "capability/multi-agent-success"
    );
    expect(multi.outcome).toBe("success");
    expect(multi.acceptance_status).toBe("accepted");

    const scoped = output.results.find(
      (r: { name: string }) =>
        r.name === "regression/success-standard-scoped-evidence"
    );
    expect(scoped.outcome).toBe("success");
    expect(scoped.acceptance_status).toBe("accepted");

    const blockedScope = output.results.find(
      (r: { name: string }) =>
        r.name === "regression/blocked-missing-evidence-scope"
    );
    expect(blockedScope.outcome).toBe("failed");
    expect(blockedScope.acceptance_status).toBe("withheld");

    const approval = output.results.find(
      (r: { name: string }) => r.name === "capability/deep-approval-required"
    );
    expect(approval.outcome).toBe("failed");
    expect(approval.acceptance_status).toBe("withheld");

    const recovery = output.results.find(
      (r: { name: string }) =>
        r.name === "capability/failed-typecheck-recovery-route"
    );
    expect(recovery.outcome).toBe("failed");
    expect(recovery.acceptance_status).toBe("withheld");

    const missingChecklist = output.results.find(
      (r: { name: string }) =>
        r.name === "adversarial/blocked-missing-done-checklist"
    );
    expect(missingChecklist.outcome).toBe("failed");
    expect(missingChecklist.acceptance_status).toBe("withheld");

    const approvalMissing = output.results.find(
      (r: { name: string }) =>
        r.name === "adversarial/standard-approval-missing"
    );
    expect(approvalMissing.outcome).toBe("failed");
    expect(approvalMissing.acceptance_status).toBe("withheld");

    const approvalPresent = output.results.find(
      (r: { name: string }) =>
        r.name === "adversarial/standard-approval-present"
    );
    expect(approvalPresent.outcome).toBe("success");
    expect(approvalPresent.acceptance_status).toBe("accepted");

    const weakPrediction = output.results.find(
      (r: { name: string }) => r.name === "regression/blocked-weak-prediction"
    );
    expect(weakPrediction.outcome).toBe("failed");
    expect(weakPrediction.acceptance_status).toBe("withheld");

    const contextAlignment = output.results.find(
      (r: { name: string }) => r.name === "regression/success-context-alignment"
    );
    expect(contextAlignment.outcome).toBe("success");
    expect(contextAlignment.acceptance_status).toBe("accepted");

    const missingContextRef = output.results.find(
      (r: { name: string }) =>
        r.name === "regression/blocked-missing-context-ref"
    );
    expect(missingContextRef.outcome).toBe("success");
    expect(missingContextRef.acceptance_status).toBe("accepted");

    const blockedContractOracle = output.results.find(
      (r: { name: string }) => r.name === "regression/blocked-contract-oracle"
    );
    expect(blockedContractOracle.outcome).toBe("success");
    expect(blockedContractOracle.acceptance_status).toBe("accepted");

    const recoveredFlow = output.results.find(
      (r: { name: string }) => r.name === "regression/success-recovered-flow"
    );
    expect(recoveredFlow.outcome).toBe("success");
    expect(recoveredFlow.acceptance_status).toBe("accepted");
  });
});
