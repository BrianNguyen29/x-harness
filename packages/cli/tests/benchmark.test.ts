import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("benchmark command", () => {
  it("reports latency for a selected command as JSON", async () => {
    const { stdout, exitCode } = await execaNode([
      "benchmark",
      "--commands",
      "verify",
      "--iterations",
      "2",
      "--filter",
      "latency",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.ok).toBe(true);
    expect(parsed.results).toHaveLength(1);
    expect(parsed.results[0].command).toBe("verify");
    expect(parsed.results[0].iterations).toBe(2);
    expect(typeof parsed.results[0].avg_ms).toBe("number");
    expect(parsed.metrics.runtime_ms).toBeGreaterThanOrEqual(
      parsed.results[0].avg_ms * parsed.results[0].iterations - 1
    );
    expect(parsed.metrics.false_accept_count).toBe(0);
  });

  it("rejects unknown benchmark commands", async () => {
    const { stderr, exitCode } = await execaNode([
      "benchmark",
      "--commands",
      "verify,unknown",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("Unknown benchmark command");
  });

  it("accepts the fast test profile command without running latency work", async () => {
    const { stdout, exitCode } = await execaNode([
      "benchmark",
      "--commands",
      "test:fast",
      "--filter",
      "admission",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.ok).toBe(true);
    expect(parsed.results).toHaveLength(0);
    expect(parsed.integration.golden.cases_total).toBeGreaterThan(0);
  });

  it("prints markdown latency report", async () => {
    const { stdout, exitCode } = await execaNode([
      "benchmark",
      "--commands",
      "verify",
    ]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("# x-harness Latency Report");
    expect(stdout).toContain("# x-harness Integration Benchmark");
    expect(stdout).toContain("| verify |");
  });

  it("runs admission benchmark fixtures without false accepts", async () => {
    const { stdout, exitCode } = await execaNode([
      "benchmark",
      "--filter",
      "admission",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.ok).toBe(true);
    expect(parsed.results).toHaveLength(0);
    expect(parsed.integration.golden.cases_total).toBeGreaterThan(0);
    expect(parsed.metrics.false_accept_count).toBe(0);
    expect(parsed.metrics.expected_pass_count).toBeGreaterThan(0);
    expect(parsed.metrics.expected_block_count).toBeGreaterThan(0);
  });

  it("runs adversarial benchmark fixtures and blocks dangerous cases", async () => {
    const { stdout, exitCode } = await execaNode([
      "benchmark",
      "--filter",
      "adversarial",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.ok).toBe(true);
    expect(parsed.integration.adversarial.cases_total).toBeGreaterThan(0);
    expect(parsed.metrics.adversarial_false_accept_count).toBe(0);
    expect(parsed.metrics.adversarial_block_rate).toBe(1);
    expect(parsed.metrics.mutation_guard_detection_rate).toBe(1);
    expect(parsed.metrics.permission_violation_detection_rate).toBe(1);
    expect(parsed.metrics.authority_violation_detection_rate).toBe(1);
    const dangerousCase = parsed.integration.adversarial.cases.find(
      (item: { name: string }) => item.name === "hidden-dangerous-command"
    );
    expect(dangerousCase.permission_violation_detected).toBe(true);
    expect(dangerousCase.errors.join("\n")).toContain(
      "permission benchmark blocked command"
    );
    const authorityCase = parsed.integration.adversarial.cases.find(
      (item: { name: string }) => item.name === "spoofed-protected-approval"
    );
    expect(authorityCase.authority_violation_detected).toBe(true);
    expect(authorityCase.errors.join("\n")).toContain(
      "governance permission violation"
    );
    const mutationCase = parsed.integration.adversarial.cases.find(
      (item: { name: string }) => item.name === "verifier-mutates-source"
    );
    expect(mutationCase.mutation_guard_detected).toBe(true);
    expect(mutationCase.blocking_predicate).toBe("verifier_not_read_only");
  });

  it("benchmarks mutation guard git and non-git fallback snapshots", async () => {
    const { stdout, exitCode } = await execaNode([
      "benchmark",
      "--filter",
      "mutation-guard",
      "--mutation-files",
      "3",
      "--mutation-concurrency",
      "1,2",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.ok).toBe(true);
    expect(parsed.results).toHaveLength(0);
    expect(parsed.integration).toBeNull();
    expect(parsed.mutation_guard_benchmark.ok).toBe(true);
    expect(parsed.mutation_guard_benchmark.file_counts).toEqual([3]);
    expect(parsed.mutation_guard_benchmark.concurrency).toEqual([1, 2]);
    expect(parsed.mutation_guard_benchmark.cases).toHaveLength(4);
    expect(
      parsed.mutation_guard_benchmark.cases.every(
        (item: { hashed_paths: number }) => item.hashed_paths === 3
      )
    ).toBe(true);
    expect(
      parsed.mutation_guard_benchmark.cases.map(
        (item: { mode: string }) => item.mode
      )
    ).toContain("non-git");
  });

  it("includes gated flag when --gate is passed", async () => {
    const { stdout, exitCode } = await execaNode([
      "benchmark",
      "--filter",
      "adversarial",
      "--gate",
      "--json",
    ]);
    expect(exitCode).toBe(0);
    const parsed = JSON.parse(stdout);
    expect(parsed.gated).toBe(true);
    expect(parsed.ok).toBe(true);
  });

  it("rejects benchmark snapshot updates without human approval workflow", async () => {
    const { stderr, exitCode } = await execaNode([
      "benchmark",
      "--update-snapshots",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("human-approved boundary change");
  });

  it("rejects empty command lists", async () => {
    const { stderr, exitCode } = await execaNode([
      "benchmark",
      "--commands",
      ",",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("--commands must include at least one command");
  });

  it("rejects malformed iterations and timeouts", async () => {
    const badIterations = await execaNode([
      "benchmark",
      "--commands",
      "verify",
      "--iterations",
      "1.5",
    ]);
    expect(badIterations.exitCode).toBe(2);
    expect(badIterations.stderr).toContain(
      "--iterations must be a positive integer"
    );

    const badTimeout = await execaNode([
      "benchmark",
      "--commands",
      "verify",
      "--timeout-ms",
      "1abc",
    ]);
    expect(badTimeout.exitCode).toBe(2);
    expect(badTimeout.stderr).toContain(
      "--timeout-ms must be a positive integer"
    );
  });

  it("rejects malformed mutation guard benchmark options", async () => {
    const { stderr, exitCode } = await execaNode([
      "benchmark",
      "--filter",
      "mutation-guard",
      "--mutation-files",
      "0,nope",
      "--json",
    ]);
    expect(exitCode).toBe(2);
    expect(stderr).toContain("--mutation-files");
  });
});
