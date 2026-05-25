import { Command } from "commander";
import {
  checkPermission,
  runPermissionFixtures,
  type PermissionDecision,
} from "../core/permissions.js";
import { CliError } from "../core/exit.js";

interface PermissionOptions {
  role?: string;
  tier?: string;
  command?: string;
  capability?: string;
  intervention?: string;
  root?: string;
  json?: boolean;
}

function renderDecisionText(decision: PermissionDecision): void {
  console.log(`# x-harness Permission ${decision.status}`);
  console.log(`- ok: ${decision.ok}`);
  console.log(`- role: ${decision.role}`);
  console.log(`- tier: ${decision.tier}`);
  if (decision.command) console.log(`- command: ${decision.command}`);
  if (decision.capability) {
    console.log(`- capability: ${decision.capability}`);
  }
  console.log(`- reason: ${decision.reason}`);
  if (decision.matched.command_set) {
    console.log(`- command_set: ${decision.matched.command_set}`);
    console.log(`- rule: ${decision.matched.rule}`);
  }
  if (decision.intervention.provided) {
    console.log(`- intervention_valid: ${decision.intervention.valid}`);
    console.log(`- intervention_reason: ${decision.intervention.reason}`);
  }
}

function normalizeOptions(opts: PermissionOptions): PermissionOptions {
  if (!opts.role) throw new CliError("--role is required", 2);
  if (opts.command && opts.capability) {
    throw new CliError("provide only one of --command or --capability", 2);
  }
  if (!opts.command && !opts.capability) {
    throw new CliError("--command or --capability is required", 2);
  }
  return opts;
}

async function permissionDecision(opts: PermissionOptions) {
  const normalized = normalizeOptions(opts);
  return checkPermission({
    root: normalized.root,
    role: normalized.role as string,
    tier: normalized.tier ?? "standard",
    command: normalized.command,
    capability: normalized.capability,
    intervention: normalized.intervention,
  });
}

export async function permissionsCheckAction(
  opts: PermissionOptions
): Promise<void> {
  const decision = await permissionDecision(opts);
  if (opts.json) {
    console.log(JSON.stringify(decision, null, 2));
  } else {
    renderDecisionText(decision);
  }
  process.exit(decision.ok ? 0 : 1);
}

export async function permissionsExplainAction(
  opts: PermissionOptions
): Promise<void> {
  const decision = await permissionDecision(opts);
  if (opts.json) {
    console.log(JSON.stringify(decision, null, 2));
  } else {
    renderDecisionText(decision);
  }
  process.exit(0);
}

export async function permissionsTestFixturesAction(opts: {
  root?: string;
  json?: boolean;
}): Promise<void> {
  const result = await runPermissionFixtures(opts.root);
  if (opts.json) {
    console.log(JSON.stringify(result, null, 2));
  } else {
    console.log("# x-harness Permission Fixtures");
    for (const fixture of result.fixtures) {
      console.log(
        `- ${fixture.ok ? "pass" : "fail"} ${fixture.name}: expected ${fixture.expected_status}, got ${fixture.actual_status}`
      );
    }
  }
  process.exit(result.ok ? 0 : 1);
}

export function permissionsCommand(): Command {
  const cmd = new Command("permissions").description(
    "Evaluate command and capability permissions without executing commands"
  );

  cmd
    .command("check")
    .description("Check whether a role/tier may use a command or capability")
    .option("--role <role>", "Role to evaluate, e.g. worker or verifier")
    .option("--tier <tier>", "Tier to evaluate", "standard")
    .option("--command <command>", "Command string to evaluate")
    .option("--capability <capability>", "Capability to evaluate")
    .option("--intervention <path>", "Intervention artifact for exceptions")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .action(permissionsCheckAction);

  cmd
    .command("explain")
    .description("Explain the permission decision for a command or capability")
    .option("--role <role>", "Role to evaluate, e.g. worker or verifier")
    .option("--tier <tier>", "Tier to evaluate", "standard")
    .option("--command <command>", "Command string to evaluate")
    .option("--capability <capability>", "Capability to evaluate")
    .option("--intervention <path>", "Intervention artifact for exceptions")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .action(permissionsExplainAction);

  cmd
    .command("test-fixtures")
    .description("Run built-in permission policy fixtures")
    .option("--root <path>", "Repository root", process.cwd())
    .option("--json", "Output JSON instead of text", false)
    .action(permissionsTestFixturesAction);

  return cmd;
}
