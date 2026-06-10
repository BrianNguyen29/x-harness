import { createHash } from "node:crypto";
import * as path from "node:path";
import { readYamlOrJson } from "./schema.js";

interface EvidenceFloorTier {
  required: readonly string[];
  oneOf?: readonly string[];
  recommended?: readonly string[];
  runtimeEnforced?: readonly string[];
}

export interface RuntimeContract {
  rules: readonly string[];
  sourceOfTruthOrder: readonly string[];
  schemaRequiredFields: readonly string[];
  successPredicates: readonly string[];
  outcomes: readonly string[];
  tiers: readonly string[];
  invalidTierLabels: readonly string[];
  evidenceFloor: {
    light: EvidenceFloorTier;
    standard: EvidenceFloorTier;
    deep: EvidenceFloorTier;
  };
  strictProvenance: readonly string[];
  fixStatus: {
    completionCard: string;
    subagentReturn: string;
  };
  generatedFrom?: readonly string[];
}

export const CANONICAL_CONTRACT = {
  rules: [
    "Completion is admitted, not claimed.",
    "Verifier is read-only.",
    "Success is the only accepted outcome.",
    "Canonical tiers: light, standard, deep.",
    "PGV is advisory-only.",
  ],
  sourceOfTruthOrder: [
    "AGENTS.md (managed block)",
    "X_HARNESS.md",
    "policies/admission.yaml",
    "policies/recovery.yaml",
    "policies/intake.yaml",
    "schemas/completion-card.schema.json",
  ],
  schemaRequiredFields: [
    "schema_version",
    "task_id",
    "tier",
    "owner",
    "accountable",
    "claim",
    "verification",
    "admission",
    "acceptance_status",
    "handoff",
  ],
  successPredicates: [
    "claim.fix_status == fixed",
    "verification.status == passed",
    "admission.outcome == success",
    "acceptance_status == accepted",
    "claim.evidence present and non-empty",
    "owner.present == true",
    "accountable.present == true",
    "evidence_floor_met",
    "admission_mapping_valid",
    "no_unresolved_blocker",
    "no_active_recovery",
    "verifier_read_only",
    "done_checklist_present_for_standard_or_deep",
    "prediction_present_for_standard_or_deep",
  ],
  outcomes: ["success", "failed", "blocked", "skipped", "timeout", "error"],
  tiers: ["light", "standard", "deep"],
  invalidTierLabels: ["small", "medium", "large"],
  evidenceFloor: {
    light: {
      required: ["files_changed"],
      oneOf: ["command_evidence", "manual_rationale"],
      recommended: [],
    },
    standard: {
      required: [
        "files_changed",
        "command_evidence",
        "done_checklist",
        "prediction",
      ],
      recommended: ["evidence_scope_declared", "untested_regions_declared"],
    },
    deep: {
      required: [
        "files_changed",
        "command_evidence",
        "evidence_scope_declared",
        "untested_regions_declared",
        "remaining_risks_declared",
        "execution_controls_present",
        "rollback_policy_present",
        "done_checklist",
        "prediction",
      ],
      runtimeEnforced: [
        "verification_artifacts",
        "state.read_set",
        "state.write_set",
      ],
    },
  },
  strictProvenance: [
    "verify --strict requires command_evidence entries to include command, exit_code, runner, and started_at for standard/deep cards.",
    "verify --strict requires verification_artifacts entries to include command, exit_code, runner, and started_at for standard/deep cards.",
  ],
  fixStatus: {
    completionCard:
      "Completion cards use claim.fix_status as the canonical fix-status field.",
    subagentReturn:
      "Subagent returns may use result.fix_status only in compatibility return payloads.",
  },
} as const;

function toStringArray(value: unknown, fallback: readonly string[]): string[] {
  if (!Array.isArray(value)) return [...fallback];
  return value.filter((item): item is string => typeof item === "string");
}

function normalizeEvidenceTier(
  value: unknown,
  fallback: (typeof CANONICAL_CONTRACT.evidenceFloor)[keyof typeof CANONICAL_CONTRACT.evidenceFloor]
): EvidenceFloorTier {
  const tier = value as Record<string, unknown> | undefined;
  return {
    required: toStringArray(tier?.required, fallback.required),
    oneOf: toStringArray(
      tier?.one_of,
      "oneOf" in fallback ? fallback.oneOf : []
    ),
    recommended: toStringArray(
      tier?.recommended,
      "recommended" in fallback ? fallback.recommended : []
    ),
    runtimeEnforced: toStringArray(
      tier?.runtime_enforced,
      "runtimeEnforced" in fallback ? fallback.runtimeEnforced : []
    ),
  };
}

async function resolveRuntimeContractRoot(root: string): Promise<string> {
  let current = path.resolve(root);
  while (true) {
    const policyPath = path.join(current, "policies", "admission.yaml");
    const schemaPath = path.join(
      current,
      "schemas",
      "completion-card.schema.json"
    );
    if ((await readExists(policyPath)) && (await readExists(schemaPath))) {
      return current;
    }
    const parent = path.dirname(current);
    if (parent === current) return path.resolve(root);
    current = parent;
  }
}

async function readExists(filePath: string): Promise<boolean> {
  try {
    await readYamlOrJson(filePath);
    return true;
  } catch {
    return false;
  }
}

export async function loadRuntimeContract(
  root = process.cwd()
): Promise<RuntimeContract> {
  const resolvedRoot = await resolveRuntimeContractRoot(root);
  const policyPath = path.join(resolvedRoot, "policies", "admission.yaml");
  const schemaPath = path.join(
    resolvedRoot,
    "schemas",
    "completion-card.schema.json"
  );
  const policy = (await readYamlOrJson(policyPath)) as Record<string, unknown>;
  const schema = (await readYamlOrJson(schemaPath)) as Record<string, unknown>;
  const evidenceFloor = policy.evidence_floor as
    | Record<string, unknown>
    | undefined;
  const outcomeMapping = policy.outcome_mapping as
    | Record<string, unknown>
    | undefined;

  return {
    ...CANONICAL_CONTRACT,
    rules: [...CANONICAL_CONTRACT.rules],
    sourceOfTruthOrder: [...CANONICAL_CONTRACT.sourceOfTruthOrder],
    schemaRequiredFields: toStringArray(
      schema.required,
      CANONICAL_CONTRACT.schemaRequiredFields
    ),
    successPredicates: toStringArray(
      policy.success_requires,
      CANONICAL_CONTRACT.successPredicates
    ),
    outcomes:
      outcomeMapping && typeof outcomeMapping === "object"
        ? Object.keys(outcomeMapping)
        : [...CANONICAL_CONTRACT.outcomes],
    tiers: [...CANONICAL_CONTRACT.tiers],
    invalidTierLabels: [...CANONICAL_CONTRACT.invalidTierLabels],
    evidenceFloor: {
      light: normalizeEvidenceTier(
        evidenceFloor?.light,
        CANONICAL_CONTRACT.evidenceFloor.light
      ),
      standard: normalizeEvidenceTier(
        evidenceFloor?.standard,
        CANONICAL_CONTRACT.evidenceFloor.standard
      ),
      deep: normalizeEvidenceTier(
        evidenceFloor?.deep,
        CANONICAL_CONTRACT.evidenceFloor.deep
      ),
    },
    strictProvenance: [...CANONICAL_CONTRACT.strictProvenance],
    fixStatus: { ...CANONICAL_CONTRACT.fixStatus },
    generatedFrom: [
      "policies/admission.yaml",
      "schemas/completion-card.schema.json",
      "packages/cli/src/core/contract.ts",
    ],
  };
}

export function getContractHash(text: string): string {
  return createHash("sha256").update(text, "utf-8").digest("hex").slice(0, 16);
}

export function renderCanonicalContext(verbose = false): string {
  if (!verbose) {
    return CANONICAL_CONTRACT.rules.join("\n");
  }
  return [
    "# x-harness Canonical Context",
    "",
    ...CANONICAL_CONTRACT.rules.map((r) => `- ${r}`),
    "",
    "## Source-of-Truth Reading Order",
    "",
    "The managed context block in AGENTS.md is authoritative. Files are read in this order:",
    "",
    ...CANONICAL_CONTRACT.sourceOfTruthOrder.map((f) => `1. ${f}`),
    "",
    "## Rules",
    "",
    "### Completion is admitted, not claimed",
    "Agents may propose completion but cannot self-admit. A completion card with `claim.fix_status: fixed` is only a completion candidate. Compatibility subagent returns may use `result.fix_status`.",
    "",
    "### Verifier is read-only",
    "The verifier may inspect files, evidence, diffs, and trace events. It must not edit source files or repair the work product while verifying.",
    "",
    "### Success is the only accepted outcome",
    "`admission.outcome: success` and `acceptance_status: accepted` are required for admission. All other outcomes are withheld.",
    "",
    "### Canonical tiers",
    "Use only `light`, `standard`, and `deep`. Do not use `small`, `medium`, or `large` in active runtime handoffs.",
    "",
    "### PGV is advisory-only",
    "Pre-gate validation (PGV) advice never overrides the verify gate and never grants admission authority by default.",
  ].join("\n");
}

export function renderCompactContextHeader(): string {
  return [
    "## Context",
    "",
    ...CANONICAL_CONTRACT.rules.map((r) => `- ${r}`),
    "",
    "For full context run: `node packages/cli/dist/index.js context`",
    "",
  ].join("\n");
}

export function renderEvidenceFloorMarkdown(
  contract: RuntimeContract = CANONICAL_CONTRACT
): string {
  const floor = contract.evidenceFloor;
  const lightOneOf = floor.light.oneOf ?? [];
  const standardRequired = floor.standard.required;
  const deepRuntime = floor.deep.runtimeEnforced ?? [];
  return [
    "## Evidence Floor",
    "",
    `- **light**: ${floor.light.required.join(" + ")} + (${lightOneOf.join(" or ")}).`,
    `- **standard**: ${standardRequired.join(" + ")}.`,
    `- **deep**: ${floor.deep.required.join(" + ")}. Runtime-enforced: ${deepRuntime.join(", ")}.`,
  ].join("\n");
}

export function renderStrictProvenanceMarkdown(
  contract: RuntimeContract = CANONICAL_CONTRACT
): string {
  return [
    "## Strict Evidence Provenance",
    "",
    ...contract.strictProvenance.map((rule) => `- ${rule}`),
  ].join("\n");
}

export function renderFixStatusGuidance(): string {
  return [
    CANONICAL_CONTRACT.fixStatus.completionCard,
    CANONICAL_CONTRACT.fixStatus.subagentReturn,
  ].join(" ");
}

export function renderRuntimeContractMarkdown(
  contract: RuntimeContract = CANONICAL_CONTRACT
): string {
  return [
    "# x-harness Generated Runtime Contract",
    "",
    "Generated from file-first source artifacts and the renderer mirror:",
    "",
    ...(contract.generatedFrom ?? CANONICAL_CONTRACT.sourceOfTruthOrder).map(
      (source) => `- ${source}`
    ),
    "",
    "## Canonical Rules",
    "",
    ...contract.rules.map((rule) => `- ${rule}`),
    "",
    "## Fix Status Fields",
    "",
    renderFixStatusGuidance(),
    "",
    "## Completion Candidate",
    "",
    "```yaml",
    "claim:",
    "  fix_status: fixed",
    "verification:",
    "  status: passed",
    "```",
    "",
    "## Accepted Completion",
    "",
    "```yaml",
    "admission:",
    "  outcome: success",
    "acceptance_status: accepted",
    "```",
    "",
    renderEvidenceFloorMarkdown(contract),
    "",
    renderStrictProvenanceMarkdown(contract),
  ].join("\n");
}

export interface ManagedContractTarget {
  id: string;
  path: string;
  render: (contract: RuntimeContract) => string;
}

export const MANAGED_CONTRACT_BEGIN_PREFIX =
  "<!-- BEGIN X-HARNESS MANAGED CONTRACT:";
export const MANAGED_CONTRACT_END_PREFIX =
  "<!-- END X-HARNESS MANAGED CONTRACT:";

function beginMarker(id: string): string {
  return `${MANAGED_CONTRACT_BEGIN_PREFIX} ${id} -->`;
}

function endMarker(id: string): string {
  return `${MANAGED_CONTRACT_END_PREFIX} ${id} -->`;
}

function renderAdapterContractMarkdown(
  contract: RuntimeContract = CANONICAL_CONTRACT
): string {
  return [
    "## Generated Adapter Contract",
    "",
    ...contract.rules.map((rule) => `- ${rule}`),
    "",
    renderEvidenceFloorMarkdown(contract),
    "",
    renderStrictProvenanceMarkdown(contract),
  ].join("\n");
}

function renderVerifyReportContractMarkdown(
  contract: RuntimeContract = CANONICAL_CONTRACT
): string {
  const lightOneOf = contract.evidenceFloor.light.oneOf ?? [];
  const deepRuntime = contract.evidenceFloor.deep.runtimeEnforced ?? [];
  return [
    "## Generated Verify Report Contract",
    "",
    `- Evidence floor light: ${contract.evidenceFloor.light.required.join(" + ")} + (${lightOneOf.join(" or ")}).`,
    `- Evidence floor standard: ${contract.evidenceFloor.standard.required.join(" + ")}.`,
    `- Evidence floor deep: ${contract.evidenceFloor.deep.required.join(" + ")}; runtime-enforced: ${deepRuntime.join(", ")}.`,
    ...contract.strictProvenance.map((rule) => `- Strict provenance: ${rule}`),
    "- Accepted completion requires `admission.outcome: success` and `acceptance_status: accepted`.",
  ].join("\n");
}

function renderHandoffContractMarkdown(
  contract: RuntimeContract = CANONICAL_CONTRACT
): string {
  return [
    "## Generated Handoff Contract",
    "",
    ...contract.rules.map((rule) => `- ${rule}`),
    "",
    "Required completion card fields:",
    "",
    ...contract.schemaRequiredFields.map((field) => `- ${field}`),
    "",
    renderEvidenceFloorMarkdown(contract),
    "",
    renderStrictProvenanceMarkdown(contract),
    "",
    renderFixStatusGuidance(),
  ].join("\n");
}

function managedTarget(
  id: string,
  targetPath: string,
  render: (contract: RuntimeContract) => string = renderAdapterContractMarkdown
): ManagedContractTarget {
  return { id, path: targetPath, render };
}

export const MANAGED_CONTRACT_TARGETS: ManagedContractTarget[] = [
  {
    id: "runtime-contract",
    path: "docs/RUNTIME_CONTRACT.md",
    render: renderRuntimeContractMarkdown,
  },
  {
    id: "admission-evidence-floor",
    path: "docs/ADMISSION_POLICY.md",
    render: (contract) =>
      [
        "## Generated Admission Evidence Floor",
        "",
        renderEvidenceFloorMarkdown(contract),
        "",
        renderStrictProvenanceMarkdown(contract),
        "",
        "Generated fix-status guidance:",
        "",
        renderFixStatusGuidance(),
      ].join("\n"),
  },
  {
    id: "verify-report-contract",
    path: "templates/VERIFY_REPORT.md",
    render: renderVerifyReportContractMarkdown,
  },
  managedTarget(
    "completion-card-template-contract",
    "templates/COMPLETION_CARD.md",
    renderHandoffContractMarkdown
  ),
  managedTarget(
    "subagent-light-template-contract",
    "templates/SUBAGENT_TASK_light.md",
    renderHandoffContractMarkdown
  ),
  managedTarget(
    "subagent-standard-template-contract",
    "templates/SUBAGENT_TASK_standard.md",
    renderHandoffContractMarkdown
  ),
  managedTarget(
    "subagent-deep-template-contract",
    "templates/SUBAGENT_TASK_deep.md",
    renderHandoffContractMarkdown
  ),
  managedTarget(
    "harness-change-template-contract",
    "templates/HARNESS_CHANGE_CONTRACT.md",
    renderHandoffContractMarkdown
  ),
  managedTarget("generic-adapter-contract", "adapters/generic/README.md"),
  managedTarget("generic-agents-contract", "adapters/generic/AGENTS.md"),
  managedTarget("claude-readme-contract", "adapters/claude-code/README.md"),
  managedTarget("claude-contract", "adapters/claude-code/CLAUDE.md"),
  managedTarget(
    "claude-admission-verifier-contract",
    "adapters/claude-code/agents/admission-verifier.md"
  ),
  managedTarget(
    "claude-implementation-worker-contract",
    "adapters/claude-code/agents/implementation-worker.md"
  ),
  managedTarget(
    "claude-handoff-skill-contract",
    "adapters/claude-code/skills/handoff/SKILL.md"
  ),
  managedTarget(
    "claude-verify-skill-contract",
    "adapters/claude-code/skills/verify/SKILL.md"
  ),
  managedTarget(
    "claude-recovery-skill-contract",
    "adapters/claude-code/skills/recovery/SKILL.md"
  ),
  managedTarget("cursor-readme-contract", "adapters/cursor/README.md"),
  managedTarget("cursor-rules-contract", "adapters/cursor/rules/x-harness.mdc"),
  managedTarget("opencode-readme-contract", "adapters/opencode/README.md"),
  managedTarget(
    "opencode-orchestrator-append-contract",
    "adapters/opencode/orchestrator_append.example.md"
  ),
  managedTarget(
    "opencode-verify-agent-redirect-contract",
    "adapters/opencode/verify-agent.md"
  ),
  managedTarget(
    "opencode-verify-agent-contract",
    "adapters/opencode/agents/x-harness-verify.md"
  ),
  managedTarget(
    "opencode-recover-agent-contract",
    "adapters/opencode/agents/x-harness-recover.md"
  ),
  managedTarget(
    "antigravity-readme-contract",
    "adapters/antigravity/README.md"
  ),
  managedTarget(
    "antigravity-rules-contract",
    "adapters/antigravity/rules/x-harness.md"
  ),
  managedTarget(
    "antigravity-implementation-workflow-contract",
    "adapters/antigravity/workflows/x-harness-implementation.md"
  ),
  managedTarget(
    "antigravity-verify-workflow-contract",
    "adapters/antigravity/workflows/x-harness-verify.md"
  ),
  managedTarget(
    "antigravity-recover-workflow-contract",
    "adapters/antigravity/workflows/x-harness-recover.md"
  ),
  managedTarget("codex-readme-contract", "adapters/codex/README.md"),
  managedTarget("codex-agents-contract", "adapters/codex/AGENTS.md"),
];

export function generateManagedContractBlock(
  target: ManagedContractTarget,
  contract: RuntimeContract = CANONICAL_CONTRACT
): string {
  const body = target.render(contract).trim();
  const hash = getContractHash(body);
  return [
    beginMarker(target.id),
    "<!-- generated-by: x-harness -->",
    `<!-- contract-hash: ${hash} -->`,
    "",
    body,
    "",
    endMarker(target.id),
  ].join("\n");
}

export function injectManagedContractBlock(
  content: string,
  target: ManagedContractTarget,
  block: string
): string {
  const begin = beginMarker(target.id);
  const end = endMarker(target.id);
  const beginIndex = content.indexOf(begin);
  const endIndex = content.indexOf(end);
  if (beginIndex !== -1 && endIndex !== -1 && endIndex > beginIndex) {
    const before = content.slice(0, beginIndex);
    const after = content.slice(endIndex + end.length);
    return before + block + after;
  }
  const separator = content.endsWith("\n") ? "\n" : "\n\n";
  return `${content}${separator}${block}\n`;
}

export function extractManagedContractBlock(
  content: string,
  target: ManagedContractTarget
): string | null {
  const begin = beginMarker(target.id);
  const end = endMarker(target.id);
  const beginIndex = content.indexOf(begin);
  const endIndex = content.indexOf(end);
  if (beginIndex === -1 || endIndex === -1 || endIndex < beginIndex) {
    return null;
  }
  return content.slice(beginIndex, endIndex + end.length);
}

export function validateManagedContractBlock(
  content: string,
  target: ManagedContractTarget,
  contract: RuntimeContract = CANONICAL_CONTRACT
): { valid: boolean; note: string } {
  const block = extractManagedContractBlock(content, target);
  if (!block) {
    return {
      valid: false,
      note: `${target.path} missing managed contract block ${target.id}`,
    };
  }
  const hashMatch = block.match(/<!-- contract-hash: ([a-f0-9]+) -->/);
  if (!hashMatch) {
    return {
      valid: false,
      note: `${target.path} managed contract block ${target.id} missing contract-hash`,
    };
  }
  const expectedBody = target.render(contract).trim();
  const expectedHash = getContractHash(expectedBody);
  const actualBody = block
    .split("\n")
    .filter((line) => {
      const trimmed = line.trim();
      return (
        trimmed !== beginMarker(target.id) &&
        trimmed !== endMarker(target.id) &&
        !trimmed.startsWith("<!--")
      );
    })
    .join("\n")
    .trim();
  if (hashMatch[1] !== expectedHash) {
    return {
      valid: false,
      note: `${target.path} managed contract block ${target.id} stale: expected ${expectedHash}, found ${hashMatch[1]}`,
    };
  }
  if (actualBody !== expectedBody) {
    return {
      valid: false,
      note: `${target.path} managed contract block ${target.id} body differs from generated contract`,
    };
  }
  return {
    valid: true,
    note: `${target.path} managed contract block ${target.id} is fresh`,
  };
}
