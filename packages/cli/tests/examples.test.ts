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
    expect(output.total).toBe(5);
    expect(output.passed).toBe(5);
    expect(output.failed).toBe(0);
    expect(output.results).toHaveLength(5);

    const names = output.results.map((r: { name: string }) => r.name);
    expect(names).toContain("success-light");
    expect(names).toContain("blocked-missing-evidence");
    expect(names).toContain("failed-invalid-status");
    expect(names).toContain("withheld-partial-fix");
    expect(names).toContain("multi-agent-success");
  });

  it("verify subcommand prints human-readable summary", async () => {
    const { stdout, exitCode } = await execaNode([
      "examples",
      "verify",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Golden examples: 5 total");
    expect(stdout).toContain("success-light");
    expect(stdout).toContain("blocked-missing-evidence");
    expect(stdout).toContain("failed-invalid-status");
    expect(stdout).toContain("withheld-partial-fix");
    expect(stdout).toContain("multi-agent-success");
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

    const successLight = output.results.find((r: { name: string }) => r.name === "success-light");
    expect(successLight.outcome).toBe("success");
    expect(successLight.acceptance_status).toBe("accepted");

    const blocked = output.results.find((r: { name: string }) => r.name === "blocked-missing-evidence");
    expect(blocked.outcome).toBe("failed");
    expect(blocked.acceptance_status).toBe("withheld");

    const failed = output.results.find((r: { name: string }) => r.name === "failed-invalid-status");
    expect(failed.outcome).toBe("failed");
    expect(failed.acceptance_status).toBe("withheld");

    const withheld = output.results.find((r: { name: string }) => r.name === "withheld-partial-fix");
    expect(withheld.outcome).toBe("failed");
    expect(withheld.acceptance_status).toBe("withheld");

    const multi = output.results.find((r: { name: string }) => r.name === "multi-agent-success");
    expect(multi.outcome).toBe("success");
    expect(multi.acceptance_status).toBe("accepted");
  });
});
