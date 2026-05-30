import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { readYamlOrJson } from "../core/schema.js";
import { runAdmission, acceptanceStatus } from "../core/admission.js";
import { validate as validateCompletionCard } from "../validators/completionCard.js";
import { resolveAssetPath } from "../core/assets.js";

interface GoldenExample {
  name: string;
  dir: string;
  cardPath: string;
  expectedOutputPath: string;
}

async function discoverGoldenExamples(): Promise<GoldenExample[]> {
  const goldenDir = await resolveAssetPath(path.join("examples", "golden"));
  if (!(await fs.pathExists(goldenDir))) {
    return [];
  }
  const examples: GoldenExample[] = [];

  async function scan(dir: string, prefix: string) {
    const entries = await fs.readdir(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (!entry.isDirectory()) continue;
      const subDir = path.join(dir, entry.name);
      const cardPath = path.join(subDir, "completion-card.yaml");
      const expectedOutputPath = path.join(
        subDir,
        "expected-verify-output.txt"
      );
      if (await fs.pathExists(cardPath)) {
        const name = prefix ? `${prefix}/${entry.name}` : entry.name;
        examples.push({ name, dir: subDir, cardPath, expectedOutputPath });
      } else {
        await scan(subDir, prefix ? `${prefix}/${entry.name}` : entry.name);
      }
    }
  }

  await scan(goldenDir, "");
  return examples.sort((a, b) => a.name.localeCompare(b.name));
}

async function verifyExample(example: GoldenExample): Promise<{
  name: string;
  passed: boolean;
  outcome: string;
  acceptanceStatus: string;
  errors: string[];
  outputMismatch?: string;
}> {
  const errors: string[] = [];
  const notes: string[] = [];

  try {
    const data = await readYamlOrJson(example.cardPath);
    const result = await validateCompletionCard(data);
    if (!result.valid) {
      errors.push(
        `completion card validation failed: ${result.errors.join("; ")}`
      );
    } else {
      notes.push(`completion card valid: ${path.basename(example.cardPath)}`);
    }

    const card = data as Record<string, unknown>;
    const admissionInput = {
      schema_version: String(card.schema_version ?? ""),
      task_id: String(card.task_id ?? ""),
      tier: (card.tier as "light" | "standard" | "deep") ?? "standard",
      owner: String(card.owner ?? ""),
      accountable: String(card.accountable ?? ""),
      claim: card.claim as Record<string, unknown>,
      verification: card.verification as Record<string, unknown>,
      admission: card.admission as Record<string, unknown>,
      acceptance_status: card.acceptance_status as "accepted" | "withheld",
      handoff: card.handoff as Record<string, unknown>,
      evidence: card.evidence as Record<string, unknown> | undefined,
      state: card.state as Record<string, unknown> | undefined,
      governance: card.governance as Record<string, unknown> | undefined,
      intake: card.intake as Record<string, unknown> | undefined,
      context_acknowledged:
        typeof card.context_acknowledged === "boolean"
          ? card.context_acknowledged
          : undefined,
      done_checklist: card.done_checklist as
        | Record<string, unknown>
        | undefined,
      prediction: card.prediction as Record<string, unknown> | undefined,
      approval_receipt: card.approval_receipt as
        | Record<string, unknown>
        | undefined,
      isCardMode: true,
      staleGround: false,
    };

    const admission = runAdmission(admissionInput);
    errors.push(...admission.errors);
    notes.push(...admission.notes);

    const outcome = errors.length > 0 ? "failed" : admission.outcome;
    const acceptance = acceptanceStatus(outcome);
    const passedChecks = notes.filter(
      (n) =>
        n.includes("valid") ||
        n.includes("passed") ||
        n.includes("checks passed")
    ).length;
    const failedChecks = errors.length;

    // Build quiet output to compare with expected
    const lines: string[] = [];
    lines.push(`outcome: ${outcome}`);
    lines.push(`acceptance_status: ${acceptance}`);
    if (failedChecks > 0) {
      lines.push(`checks: ${passedChecks} passed, ${failedChecks} failed`);
    } else {
      lines.push(`checks: ${passedChecks} passed, 0 failed`);
    }
    const actualOutput = lines.join("\n") + "\n";

    let outputMismatch: string | undefined;
    if (!(await fs.pathExists(example.expectedOutputPath))) {
      outputMismatch = `Missing expected output snapshot: ${path.relative(process.cwd(), example.expectedOutputPath)}`;
    } else {
      const expectedOutput = await fs.readFile(
        example.expectedOutputPath,
        "utf-8"
      );
      if (actualOutput.trim() !== expectedOutput.trim()) {
        outputMismatch = `Output mismatch.\nExpected:\n${expectedOutput}\nActual:\n${actualOutput}`;
      }
    }

    const passed = !outputMismatch;

    return {
      name: example.name,
      passed,
      outcome,
      acceptanceStatus: acceptance,
      errors,
      outputMismatch,
    };
  } catch (err) {
    errors.push(
      `unexpected error: ${err instanceof Error ? err.message : String(err)}`
    );
    return {
      name: example.name,
      passed: false,
      outcome: "error",
      acceptanceStatus: "withheld",
      errors,
    };
  }
}

export function examplesCommand(): Command {
  const cmd = new Command("examples")
    .description("Run x-harness against golden examples")
    .addCommand(
      new Command("verify")
        .description("Verify all golden completion cards")
        .option("--json", "Output JSON instead of human-readable text", false)
        .action(async (opts: { json?: boolean }) => {
          const examples = await discoverGoldenExamples();
          if (examples.length === 0) {
            const msg = "No golden examples found.";
            if (opts.json) {
              console.log(JSON.stringify({ ok: false, error: msg }, null, 2));
            } else {
              console.error(msg);
            }
            process.exit(1);
          }

          const results = [];
          for (const example of examples) {
            const result = await verifyExample(example);
            results.push(result);
          }

          const allPassed = results.every((r) => r.passed);
          const exitCode = allPassed ? 0 : 1;

          if (opts.json) {
            console.log(
              JSON.stringify(
                {
                  ok: allPassed,
                  total: results.length,
                  passed: results.filter((r) => r.passed).length,
                  failed: results.filter((r) => !r.passed).length,
                  results: results.map((r) => ({
                    name: r.name,
                    passed: r.passed,
                    outcome: r.outcome,
                    acceptance_status: r.acceptanceStatus,
                    errors: r.errors,
                    output_mismatch: r.outputMismatch ?? null,
                  })),
                },
                null,
                2
              )
            );
          } else {
            console.log(`Golden examples: ${results.length} total`);
            for (const r of results) {
              const icon = r.passed ? "✓" : "✗";
              console.log(
                `${icon} ${r.name}: ${r.outcome} (${r.acceptanceStatus})`
              );
              if (r.errors.length > 0) {
                for (const err of r.errors) {
                  console.log(`  - ${err}`);
                }
              }
              if (r.outputMismatch) {
                console.log(`  - ${r.outputMismatch}`);
              }
            }
            console.log("");
            console.log(
              allPassed
                ? "All golden examples passed."
                : "Some golden examples failed."
            );
          }

          process.exit(exitCode);
        })
    );
  return cmd;
}
