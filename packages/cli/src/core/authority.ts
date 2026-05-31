import * as path from "node:path";
import fs from "fs-extra";
import { readYamlOrJson } from "./schema.js";
import { sha256File } from "./hash.js";

export type AuthorityClass =
  | "agent_editable"
  | "agent_proposable_human_approved"
  | "human_only";

export interface ProtectedPath {
  path: string;
  authority: AuthorityClass;
  rationale: string;
}

export interface AuthorityPolicy {
  version: number;
  authority_classes: Record<
    string,
    { description: string; examples: string[] }
  >;
  protected_paths: ProtectedPath[];
  report_only: boolean;
  authority_enforcement?: {
    mode?: "report_only" | "enforced";
    admission_behavior?: "withhold";
    require_verified_intervention?: boolean;
  };
  governance_check: {
    behavior: string;
    exit_on_warnings: boolean;
    block_on_violations: boolean;
  };
}

function globMatch(pattern: string, filePath: string): boolean {
  // Escape regex metacharacters first, then apply glob wildcards
  const escaped = pattern.replace(/[.+^${}()|[\]\\]/g, "\\$&");
  const globPattern = escaped
    .replace(/\*\*/g, "{{DOUBLE_STAR}}")
    .replace(/\*/g, "[^/]*")
    .replace(/\{\{DOUBLE_STAR\}\}/g, ".*");

  const regex = new RegExp(`^${globPattern}$`);
  return regex.test(filePath);
}

function matchPath(pattern: string, filePath: string): boolean {
  // Normalize both paths
  const normalizedPattern = pattern.replace(/\\/g, "/");
  const normalizedFilePath = filePath.replace(/\\/g, "/");

  // Direct match
  if (globMatch(normalizedPattern, normalizedFilePath)) {
    return true;
  }

  // Directory prefix match (e.g., "packages/cli/src/**/*.ts" matches "packages/cli/src/commands/foo.ts")
  if (pattern.endsWith("/**")) {
    const dirPattern = pattern.slice(0, -3);
    if (normalizedFilePath.startsWith(dirPattern)) {
      return true;
    }
  }

  // Single-level wildcard match
  if (pattern.includes("/*/")) {
    const parts = normalizedFilePath.split("/");
    const patternParts = normalizedPattern.split("/");
    if (parts.length === patternParts.length) {
      return globMatch(normalizedPattern, normalizedFilePath);
    }
  }

  return false;
}

let cachedPolicy: AuthorityPolicy | null = null;
let policyCachePath: string | null = null;

export async function loadAuthorityPolicy(
  root?: string
): Promise<AuthorityPolicy> {
  const repoRoot = root ?? process.cwd();
  const policyPath = path.resolve(repoRoot, "policies", "authority.yaml");

  // Return cache if valid
  if (cachedPolicy && policyCachePath === policyPath) {
    return cachedPolicy;
  }

  try {
    const data = await readYamlOrJson(policyPath);
    cachedPolicy = data as AuthorityPolicy;
    policyCachePath = policyPath;
    return cachedPolicy;
  } catch (err) {
    throw new Error(
      `Failed to load authority policy: ${err instanceof Error ? err.message : String(err)}`
    );
  }
}

export function classifyPath(
  filePath: string,
  policy: AuthorityPolicy
): { authority: AuthorityClass; rationale: string } {
  const normalizedPath = filePath.replace(/\\/g, "/");

  // Check protected paths in order
  for (const protectedPath of policy.protected_paths) {
    if (matchPath(protectedPath.path, normalizedPath)) {
      return {
        authority: protectedPath.authority,
        rationale: protectedPath.rationale,
      };
    }
  }

  // Default to agent_editable if no match
  return {
    authority: "agent_editable",
    rationale: "Default: no protected path match",
  };
}

export function getProtectedPaths(policy: AuthorityPolicy): ProtectedPath[] {
  return policy.protected_paths;
}

export function explainPath(
  filePath: string,
  root?: string
): Promise<{ authority: AuthorityClass; rationale: string; path: string }> {
  return loadAuthorityPolicy(root).then((policy) => {
    const repoRoot = root ?? process.cwd();
    let normalizedPath = filePath.replace(/\\/g, "/");

    // If absolute path, convert to relative from repo root
    if (path.isAbsolute(normalizedPath)) {
      normalizedPath = path
        .relative(repoRoot, normalizedPath)
        .replace(/\\/g, "/");
    }

    const result = classifyPath(normalizedPath, policy);
    return {
      ...result,
      path: normalizedPath,
    };
  });
}

export interface GovernanceWarning {
  path: string;
  authority: AuthorityClass;
  rationale: string;
  severity: "warning" | "violation";
  approval_required: boolean;
  approval_verified: boolean;
  approval_note?: string;
}

export interface GovernanceCheckResult {
  violations: GovernanceWarning[];
  warnings: GovernanceWarning[];
  report_only: boolean;
  enforced: boolean;
  total_violations: number;
  total_warnings: number;
}

export interface GovernanceCheckOptions {
  enforce?: boolean;
  governance?: Record<string, unknown>;
}

interface ApprovalRegistryEntry {
  path?: string;
  sha256?: string;
  status?: string;
  approved_by?: string;
  scope?: {
    paths?: string[];
  };
}

function normalizedHash(value: string): string {
  return value
    .replace(/^sha256:/i, "")
    .trim()
    .toLowerCase();
}

function rootRelativePath(root: string, filePath: string): string | null {
  const resolved = path.resolve(root, filePath);
  const prefix = root.endsWith(path.sep) ? root : `${root}${path.sep}`;
  if (resolved !== root && !resolved.startsWith(prefix)) return null;
  return resolved;
}

function getApprovalArtifact(
  governance?: Record<string, unknown>
): Record<string, unknown> | null {
  const artifact = governance?.approval_artifact;
  return artifact && typeof artifact === "object"
    ? (artifact as Record<string, unknown>)
    : null;
}

function scopePathsFromRecord(record: Record<string, unknown>): string[] {
  const scope = record.scope;
  if (!scope || typeof scope !== "object") return [];
  const paths = (scope as Record<string, unknown>).paths;
  return Array.isArray(paths)
    ? paths.filter((item): item is string => typeof item === "string")
    : [];
}

async function loadApprovalRegistry(
  root: string
): Promise<ApprovalRegistryEntry[]> {
  const registryPath = path.join(
    root,
    ".x-harness",
    "approvals",
    "registry.json"
  );
  if (!(await fs.pathExists(registryPath))) return [];
  const registry = (await readYamlOrJson(registryPath)) as Record<
    string,
    unknown
  >;
  const approvals = registry.approvals;
  return Array.isArray(approvals)
    ? approvals.filter(
        (item): item is ApprovalRegistryEntry =>
          Boolean(item) && typeof item === "object"
      )
    : [];
}

async function findRegisteredApproval(input: {
  root: string;
  artifactPath: string;
  actualHash: string;
}): Promise<ApprovalRegistryEntry | null> {
  const registry = await loadApprovalRegistry(input.root);
  return (
    registry.find(
      (entry) =>
        entry.path === input.artifactPath &&
        normalizedHash(String(entry.sha256 ?? "")) === input.actualHash
    ) ?? null
  );
}

function scopeCoversPath(scopePaths: string[], protectedPath: string): boolean {
  return scopePaths.some((scopePath) => matchPath(scopePath, protectedPath));
}

async function verifyApprovalForPath(
  governance: Record<string, unknown> | undefined,
  protectedPath: string,
  root: string
): Promise<{ ok: boolean; note: string }> {
  if (governance?.approval_status !== "approved") {
    return { ok: false, note: "governance approval_status is not approved" };
  }
  const artifact = getApprovalArtifact(governance);
  if (!artifact) {
    return { ok: false, note: "governance approval_artifact is missing" };
  }
  const artifactPath = artifact.path;
  const expectedHash = artifact.sha256;
  if (typeof artifactPath !== "string" || artifactPath.trim().length === 0) {
    return { ok: false, note: "governance approval_artifact.path is missing" };
  }
  if (typeof expectedHash !== "string" || expectedHash.trim().length === 0) {
    return {
      ok: false,
      note: "governance approval_artifact.sha256 is missing",
    };
  }
  const resolvedArtifactPath = rootRelativePath(root, artifactPath);
  if (!resolvedArtifactPath) {
    return {
      ok: false,
      note: "governance approval_artifact.path is outside repository root",
    };
  }
  if (!(await fs.pathExists(resolvedArtifactPath))) {
    return { ok: false, note: "governance approval_artifact.path not found" };
  }
  const artifactRelativePath = path
    .relative(root, resolvedArtifactPath)
    .replace(/\\/g, "/");
  if (!artifactRelativePath.startsWith(".x-harness/approvals/")) {
    return {
      ok: false,
      note: "governance approval_artifact.path must be under .x-harness/approvals",
    };
  }
  const actualHash = await sha256File(resolvedArtifactPath);
  if (normalizedHash(expectedHash) !== actualHash) {
    return { ok: false, note: "governance approval_artifact.sha256 mismatch" };
  }
  const registeredApproval = await findRegisteredApproval({
    root,
    artifactPath: artifactRelativePath,
    actualHash,
  });
  if (!registeredApproval) {
    return {
      ok: false,
      note: "approval artifact is not registered in .x-harness/approvals/registry.json",
    };
  }
  if (registeredApproval.status !== "approved") {
    return { ok: false, note: "registered approval status is not approved" };
  }
  const approvalRecord = (await readYamlOrJson(resolvedArtifactPath)) as Record<
    string,
    unknown
  >;
  if (approvalRecord.decision !== "approved") {
    return { ok: false, note: "approval artifact decision is not approved" };
  }
  if (
    typeof approvalRecord.approved_by !== "string" ||
    approvalRecord.approved_by.trim().length === 0
  ) {
    return { ok: false, note: "approval artifact approved_by is missing" };
  }
  if (
    typeof registeredApproval.approved_by === "string" &&
    registeredApproval.approved_by !== approvalRecord.approved_by
  ) {
    return {
      ok: false,
      note: "registered approval approver does not match approval artifact",
    };
  }
  if (
    typeof approvalRecord.approved_at !== "string" ||
    approvalRecord.approved_at.trim().length === 0
  ) {
    return { ok: false, note: "approval artifact approved_at is missing" };
  }
  const scopePaths = scopePathsFromRecord(approvalRecord);
  if (!scopeCoversPath(scopePaths, protectedPath)) {
    return {
      ok: false,
      note: "approval artifact scope does not cover protected path",
    };
  }
  const registeredScopePaths = scopePathsFromRecord(
    registeredApproval as Record<string, unknown>
  );
  if (
    registeredScopePaths.length > 0 &&
    !scopeCoversPath(registeredScopePaths, protectedPath)
  ) {
    return {
      ok: false,
      note: "registered approval scope does not cover protected path",
    };
  }
  return { ok: true, note: "approval artifact verified" };
}

function isEnforced(
  policy: AuthorityPolicy,
  options?: GovernanceCheckOptions
): boolean {
  return (
    options?.enforce === true ||
    policy.authority_enforcement?.mode === "enforced"
  );
}

export async function checkGovernance(
  files: string[],
  root?: string,
  options?: GovernanceCheckOptions
): Promise<GovernanceCheckResult> {
  const policy = await loadAuthorityPolicy(root);
  const repoRoot = root ?? process.cwd();
  const enforced = isEnforced(policy, options);
  const warnings: GovernanceWarning[] = [];
  const violations: GovernanceWarning[] = [];

  for (const file of files) {
    let normalizedPath = file.replace(/\\/g, "/");

    // If absolute path, convert to relative from repo root
    if (path.isAbsolute(normalizedPath)) {
      normalizedPath = path
        .relative(repoRoot, normalizedPath)
        .replace(/\\/g, "/");
    }

    const { authority, rationale } = classifyPath(normalizedPath, policy);

    if (authority === "human_only") {
      const approval = enforced
        ? await verifyApprovalForPath(
            options?.governance,
            normalizedPath,
            repoRoot
          )
        : { ok: false, note: "report-only mode" };
      const finding: GovernanceWarning = {
        path: normalizedPath,
        authority,
        rationale,
        severity: approval.ok ? "warning" : enforced ? "violation" : "warning",
        approval_required: true,
        approval_verified: approval.ok,
        approval_note: approval.note,
      };
      if (enforced && !approval.ok) violations.push(finding);
      else if (!approval.ok) warnings.push(finding);
    } else if (authority === "agent_proposable_human_approved") {
      const approval = enforced
        ? await verifyApprovalForPath(
            options?.governance,
            normalizedPath,
            repoRoot
          )
        : { ok: false, note: "report-only mode" };
      const finding: GovernanceWarning = {
        path: normalizedPath,
        authority,
        rationale,
        severity: approval.ok ? "warning" : enforced ? "violation" : "warning",
        approval_required: true,
        approval_verified: approval.ok,
        approval_note: approval.note,
      };
      if (enforced && !approval.ok) violations.push(finding);
      else if (!approval.ok) warnings.push(finding);
    }
  }

  return {
    violations,
    warnings,
    report_only: !enforced,
    enforced,
    total_violations: violations.length,
    total_warnings: warnings.length,
  };
}

export function isReportOnly(policy: AuthorityPolicy): boolean {
  return policy.report_only ?? true;
}
