import * as path from "node:path";
import fs from "fs-extra";
import { compileSchema, loadSchema, readYamlOrJson } from "./schema.js";

export interface CommandSet {
  allow?: string[];
  allow_patterns?: string[];
  deny?: string[];
  deny_patterns?: string[];
}

export interface PermissionProfile {
  allow_capabilities?: string[];
  deny_capabilities?: string[];
  require_approval?: string[];
  allow_command_sets?: string[];
  deny_command_sets?: string[];
}

export interface PermissionsPolicy {
  version: number;
  command_sets: Record<string, CommandSet>;
  roles: Record<string, Record<string, PermissionProfile>>;
}

export interface PermissionCheckInput {
  root?: string;
  role: string;
  tier?: string;
  command?: string;
  capability?: string;
  intervention?: string;
}

export interface PermissionDecision {
  ok: boolean;
  status: "allowed" | "denied" | "requires_intervention";
  role: string;
  tier: string;
  command: string | null;
  capability: string | null;
  reason: string;
  matched: {
    command_set: string | null;
    rule: string | null;
  };
  intervention: {
    provided: boolean;
    valid: boolean;
    reason: string | null;
    path: string | null;
  };
  admission_authority: false;
}

export interface PermissionFixtureResult {
  name: string;
  ok: boolean;
  expected_status: PermissionDecision["status"];
  actual_status: PermissionDecision["status"];
  decision: PermissionDecision;
}

interface InterventionArtifact {
  decision?: string;
  expiration?: string;
  scope?: string;
  paths?: string[];
}

function normalizeCommand(command: string): string {
  return command.trim().replace(/\s+/g, " ");
}

function profileFor(
  policy: PermissionsPolicy,
  role: string,
  tier: string
): PermissionProfile | null {
  const roleProfiles = policy.roles[role];
  if (!roleProfiles) return null;
  return roleProfiles[tier] ?? roleProfiles.all ?? null;
}

function listIncludes(items: string[] | undefined, item: string): boolean {
  return (items ?? []).includes(item);
}

function regexMatches(pattern: string, input: string): boolean {
  return new RegExp(pattern).test(input);
}

function commandMatches(
  command: string,
  set: CommandSet,
  mode: "allow" | "deny"
): string | null {
  const exact = mode === "allow" ? set.allow : set.deny;
  const patterns = mode === "allow" ? set.allow_patterns : set.deny_patterns;
  if ((exact ?? []).includes(command)) return command;
  for (const pattern of patterns ?? []) {
    if (regexMatches(pattern, command)) return pattern;
  }
  return null;
}

function shellMetacharacter(command: string): string | null {
  const checks: Array<[string, RegExp]> = [
    ["&&", /&&/],
    ["||", /\|\|/],
    [";", /;/],
    ["|", /\|/],
    ["`", /`/],
    ["$(", /\$\(/],
    [">", />/],
    ["<", /</],
  ];
  for (const [token, pattern] of checks) {
    if (pattern.test(command)) return token;
  }
  return null;
}

function findCommandMatch(
  policy: PermissionsPolicy,
  command: string,
  setNames: string[] | undefined,
  mode: "allow" | "deny"
): { command_set: string; rule: string } | null {
  for (const setName of setNames ?? []) {
    const set = policy.command_sets[setName];
    if (!set) {
      return { command_set: setName, rule: "unknown_command_set" };
    }
    const rule = commandMatches(command, set, mode);
    if (rule) return { command_set: setName, rule };
  }
  return null;
}

function baseDecision(
  input: Required<Pick<PermissionCheckInput, "role">> & PermissionCheckInput,
  status: PermissionDecision["status"],
  reason: string,
  matched: PermissionDecision["matched"] = {
    command_set: null,
    rule: null,
  },
  intervention: PermissionDecision["intervention"] = {
    provided: Boolean(input.intervention),
    valid: false,
    reason: input.intervention ? "not evaluated" : null,
    path: input.intervention ? path.resolve(input.intervention) : null,
  }
): PermissionDecision {
  return {
    ok: status === "allowed",
    status,
    role: input.role,
    tier: input.tier ?? "standard",
    command: input.command ? normalizeCommand(input.command) : null,
    capability: input.capability ?? null,
    reason,
    matched,
    intervention,
    admission_authority: false,
  };
}

export async function validatePermissionsPolicy(
  policy: PermissionsPolicy
): Promise<{ ok: boolean; errors: string[] }> {
  const schema = await loadSchema("permissions");
  const validate = compileSchema(schema);
  if (!validate(policy)) {
    return {
      ok: false,
      errors: (validate.errors ?? []).map(
        (err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`
      ),
    };
  }

  const errors: string[] = [];
  for (const [role, tiers] of Object.entries(policy.roles)) {
    for (const [tier, profile] of Object.entries(tiers)) {
      for (const setName of [
        ...(profile.allow_command_sets ?? []),
        ...(profile.deny_command_sets ?? []),
      ]) {
        if (!policy.command_sets[setName]) {
          errors.push(
            `${role}.${tier} references unknown command set ${setName}`
          );
        }
      }
    }
  }
  return { ok: errors.length === 0, errors };
}

export async function loadPermissionsPolicy(
  root?: string
): Promise<PermissionsPolicy> {
  const repoRoot = root ?? process.cwd();
  const policyPath = path.resolve(repoRoot, "policies", "permissions.yaml");
  const policy = (await readYamlOrJson(policyPath)) as PermissionsPolicy;
  const validation = await validatePermissionsPolicy(policy);
  if (!validation.ok) {
    throw new Error(
      `permissions policy validation failed: ${validation.errors.join("; ")}`
    );
  }
  return policy;
}

function interventionTarget(input: PermissionCheckInput): string {
  if (input.capability) return `capability:${input.capability}`;
  if (input.command) return `command:${normalizeCommand(input.command)}`;
  return "permissions";
}

function interventionCovers(
  artifact: InterventionArtifact,
  input: PermissionCheckInput
): boolean {
  if (artifact.scope === "global") return true;
  const target = interventionTarget(input);
  const paths = artifact.paths ?? [];
  return paths.some(
    (entry) =>
      entry === target ||
      entry === "permissions" ||
      entry === "permissions/**" ||
      entry === "policies/permissions.yaml"
  );
}

async function validateInterventionForPermission(
  input: PermissionCheckInput
): Promise<PermissionDecision["intervention"]> {
  if (!input.intervention) {
    return {
      provided: false,
      valid: false,
      reason: "intervention required",
      path: null,
    };
  }

  const interventionPath = path.resolve(
    input.root ?? process.cwd(),
    input.intervention
  );
  if (!(await fs.pathExists(interventionPath))) {
    return {
      provided: true,
      valid: false,
      reason: "intervention file not found",
      path: interventionPath,
    };
  }

  const artifact = (await readYamlOrJson(
    interventionPath
  )) as InterventionArtifact;
  const schema = await loadSchema("intervention");
  const validate = compileSchema(schema);
  if (!validate(artifact)) {
    return {
      provided: true,
      valid: false,
      reason: (validate.errors ?? [])
        .map((err) => `${err.instancePath || "/"} ${err.message ?? "invalid"}`)
        .join("; "),
      path: interventionPath,
    };
  }

  if (artifact.decision !== "allow" && artifact.decision !== "override") {
    return {
      provided: true,
      valid: false,
      reason: "intervention decision must be allow or override",
      path: interventionPath,
    };
  }

  const expiration = new Date(artifact.expiration ?? "").getTime();
  if (!Number.isFinite(expiration) || expiration <= Date.now()) {
    return {
      provided: true,
      valid: false,
      reason: "intervention is expired",
      path: interventionPath,
    };
  }

  if (!interventionCovers(artifact, input)) {
    return {
      provided: true,
      valid: false,
      reason: `intervention scope does not cover ${interventionTarget(input)}`,
      path: interventionPath,
    };
  }

  return {
    provided: true,
    valid: true,
    reason: "valid intervention exception",
    path: interventionPath,
  };
}

export async function checkPermission(
  input: PermissionCheckInput
): Promise<PermissionDecision> {
  if (!input.command && !input.capability) {
    return baseDecision(input, "denied", "command or capability is required");
  }
  if (input.command && input.capability) {
    return baseDecision(
      input,
      "denied",
      "provide only one of command or capability"
    );
  }

  const tier = input.tier ?? "standard";
  const policy = await loadPermissionsPolicy(input.root);
  const profile = profileFor(policy, input.role, tier);
  if (!profile) {
    return baseDecision(
      input,
      "denied",
      `no permissions profile for role ${input.role} tier ${tier}`
    );
  }

  if (input.command) {
    const command = normalizeCommand(input.command);
    const denied = findCommandMatch(
      policy,
      command,
      profile.deny_command_sets,
      "deny"
    );
    if (denied) {
      return baseDecision(
        input,
        "denied",
        `command denied by ${denied.command_set}`,
        denied
      );
    }

    const metacharacter = shellMetacharacter(command);
    if (metacharacter) {
      return baseDecision(
        input,
        "denied",
        `command contains shell metacharacter ${metacharacter}`,
        { command_set: "shell_metacharacter", rule: metacharacter }
      );
    }

    const allowed = findCommandMatch(
      policy,
      command,
      profile.allow_command_sets,
      "allow"
    );
    if (allowed) {
      return baseDecision(
        input,
        "allowed",
        `command allowed by ${allowed.command_set}`,
        allowed
      );
    }

    return baseDecision(
      input,
      "denied",
      `command is not allowlisted for role ${input.role} tier ${tier}`
    );
  }

  const capability = input.capability as string;
  if (listIncludes(profile.deny_capabilities, capability)) {
    return baseDecision(
      input,
      "denied",
      `capability ${capability} is denied for role ${input.role}`
    );
  }

  if (listIncludes(profile.require_approval, capability)) {
    const intervention = await validateInterventionForPermission(input);
    if (intervention.valid) {
      return baseDecision(
        input,
        "allowed",
        `capability ${capability} allowed by valid intervention`,
        { command_set: null, rule: "intervention" },
        intervention
      );
    }
    return baseDecision(
      input,
      "requires_intervention",
      `capability ${capability} requires valid intervention`,
      { command_set: null, rule: "require_approval" },
      intervention
    );
  }

  if (listIncludes(profile.allow_capabilities, capability)) {
    return baseDecision(
      input,
      "allowed",
      `capability ${capability} allowed for role ${input.role}`
    );
  }

  return baseDecision(
    input,
    "denied",
    `capability ${capability} is not allowed for role ${input.role} tier ${tier}`
  );
}

export async function runPermissionFixtures(root?: string): Promise<{
  ok: boolean;
  fixtures: PermissionFixtureResult[];
}> {
  const fixtures: Array<{
    name: string;
    input: PermissionCheckInput;
    expected: PermissionDecision["status"];
  }> = [
    {
      name: "worker_safe_test_allowed",
      input: { root, role: "worker", tier: "standard", command: "npm test" },
      expected: "allowed",
    },
    {
      name: "dangerous_command_denied",
      input: { root, role: "worker", tier: "deep", command: "rm -rf dist" },
      expected: "denied",
    },
    {
      name: "chained_command_denied",
      input: {
        root,
        role: "worker",
        tier: "standard",
        command: "npm test && node scripts/mutate.js",
      },
      expected: "denied",
    },
    {
      name: "verifier_write_source_denied",
      input: {
        root,
        role: "verifier",
        tier: "deep",
        capability: "write_source",
      },
      expected: "denied",
    },
    {
      name: "deep_dependency_install_requires_intervention",
      input: {
        root,
        role: "worker",
        tier: "deep",
        capability: "dependency_install",
      },
      expected: "requires_intervention",
    },
  ];

  const results: PermissionFixtureResult[] = [];
  for (const fixture of fixtures) {
    const decision = await checkPermission(fixture.input);
    results.push({
      name: fixture.name,
      ok: decision.status === fixture.expected,
      expected_status: fixture.expected,
      actual_status: decision.status,
      decision,
    });
  }
  return {
    ok: results.every((result) => result.ok),
    fixtures: results,
  };
}
