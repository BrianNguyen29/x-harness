import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { loadSchema, compileSchema, readYamlOrJson } from "../core/schema.js";
import { validateManagedBlock } from "../core/context.js";
import {
  CANONICAL_CONTRACT,
  loadRuntimeContract,
  MANAGED_CONTRACT_TARGETS,
  validateManagedContractBlock,
} from "../core/contract.js";
import { validateComponentsRegistry } from "../core/components.js";

// Known predicates/fields from the generated contract mirror. The mirror is
// checked against file-first policy/schema artifacts by policy_drift.
const KNOWN_SCHEMA_REQUIRED_FIELDS: string[] = [
  ...CANONICAL_CONTRACT.schemaRequiredFields,
];

const KNOWN_SUCCESS_PREDICATES: string[] = [
  ...CANONICAL_CONTRACT.successPredicates,
];

const KNOWN_TIER_EVIDENCE_LABELS: Record<string, string[]> = {
  light: [
    ...CANONICAL_CONTRACT.evidenceFloor.light.required,
    ...CANONICAL_CONTRACT.evidenceFloor.light.oneOf,
  ],
  standard: [
    ...CANONICAL_CONTRACT.evidenceFloor.standard.required,
    ...CANONICAL_CONTRACT.evidenceFloor.standard.recommended,
  ],
  deep: [
    ...CANONICAL_CONTRACT.evidenceFloor.deep.required,
    ...CANONICAL_CONTRACT.evidenceFloor.deep.runtimeEnforced,
  ],
};

const KNOWN_OUTCOMES: string[] = [...CANONICAL_CONTRACT.outcomes];

const KNOWN_REJECT_KEYS = [
  "claim.fix_status",
  "fix_status",
  "verification_status",
  "evidence_quality",
  "approval_required_but_missing",
  "timeout",
  "error",
];

const CRITICAL_ASSETS = [
  "README.md",
  "AGENTS.md",
  "X_HARNESS.md",
  "docs/README.md",
  "docs/QUICKSTART.md",
  "docs/FAQ.md",
  "docs/VERIFY_GATE.md",
  "docs/RUNTIME_CONTRACT.md",
  "docs/ADMISSION_POLICY.md",
  "docs/ADAPTERS.md",
  "docs/SCHEMAS.md",
  "docs/RECOVERY.md",
  "docs/PACKETS.md",
  "docs/CLEANUP.md",
  "docs/CI.md",
  "docs/ARCHITECTURE.md",
  "docs/REPORT_FORMATS.md",
  "docs/RELEASE_SECURITY.md",
  "templates/COMPLETION_CARD.md",
  "templates/SUBAGENT_TASK_light.md",
  "templates/SUBAGENT_TASK_standard.md",
  "templates/SUBAGENT_TASK_deep.md",
  "templates/HARNESS_CHANGE_CONTRACT.md",
  "components/registry.yaml",
  "schemas/attribution.schema.json",
  "schemas/agent-profile.schema.json",
  "schemas/approval-risk.schema.json",
  "schemas/benchmark-report.schema.json",
  "schemas/claim.schema.json",
  "schemas/completion-card.schema.json",
  "schemas/components-registry.schema.json",
  "schemas/cost-budget.schema.json",
  "schemas/evidence.schema.json",
  "schemas/evidence-index.schema.json",
  "schemas/evolution-constitution.schema.json",
  "schemas/episode-manifest.schema.json",
  "schemas/frozen-manifest.schema.json",
  "schemas/federation-pattern.schema.json",
  "schemas/intervention.schema.json",
  "schemas/packet.schema.json",
  "schemas/permissions.schema.json",
  "schemas/subagent-return.schema.json",
  "schemas/verify-event.schema.json",
  "schemas/pgv-advice.schema.json",
  "policies/admission.yaml",
  "policies/approval-risk.yaml",
  "policies/authority.yaml",
  "policies/cleanup.yaml",
  "policies/cost-budget.yaml",
  "policies/intake.yaml",
  "policies/permissions.yaml",
  "policies/federation.yaml",
  "policies/recovery.yaml",
  "tools/experimental/evolve/constitution.yaml",
  "tools/experimental/evolve/evolution-budget.yaml",
];

const CORE_SCHEMAS = [
  "attribution",
  "agent-profile",
  "approval-risk",
  "benchmark-report",
  "claim",
  "completion-card",
  "components-registry",
  "cost-budget",
  "evidence",
  "evidence-index",
  "evolution-constitution",
  "episode-manifest",
  "frozen-manifest",
  "federation-pattern",
  "intervention",
  "packet",
  "permissions",
  "subagent-return",
  "verify-event",
  "pgv-advice",
];

const ADAPTERS = [
  "adapters/generic",
  "adapters/claude-code",
  "adapters/cursor",
  "adapters/opencode",
  "adapters/antigravity",
];

const DANGEROUS_PGV_PHRASES = [
  "PGV blocks",
  "PGV gates",
  "PGV decides",
  "PGV is authoritative",
  "PGV overrides verify",
];

const DOCTOR_SCAN_DIRS = ["docs", "templates", "adapters"];
const DOCTOR_TIER_SCAN_DIRS = [
  ...DOCTOR_SCAN_DIRS,
  path.join("packages", "cli", "src"),
];
const DOCTOR_PGV_SCAN_DIRS = [
  ...DOCTOR_SCAN_DIRS,
  path.join("packages", "cli", "src", "core"),
];

function isDoctorTextFile(filePath: string): boolean {
  return [".md", ".mdc", ".ts", ".json"].includes(path.extname(filePath));
}

async function walkDoctorFiles(
  root: string,
  dirs: string[]
): Promise<string[]> {
  const files: string[] = [];
  for (const relDir of dirs) {
    const dir = path.join(root, relDir);
    if (!(await fs.pathExists(dir))) continue;
    const walk = async (current: string) => {
      const entries = await fs.readdir(current, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(current, entry.name);
        if (entry.isDirectory()) {
          await walk(fullPath);
        } else if (entry.isFile() && isDoctorTextFile(fullPath)) {
          files.push(fullPath);
        }
      }
    };
    await walk(dir);
  }
  return files.sort();
}

function isAllowedInvalidTierReference(line: string): boolean {
  return (
    /do not use\b.*\b(small|medium|large)\b/i.test(line) ||
    /forbidden active aliases/i.test(line) ||
    /invalid tier labels/i.test(line) ||
    /risk.*\b(medium|large|small)\b/i.test(line) ||
    /confidence.*\b(medium|large|small)\b/i.test(line) ||
    /priority.*\b(medium|large|small)\b/i.test(line) ||
    /context[_ -]?class.*\b(medium|large|small)\b/i.test(line) ||
    /default_token_impact.*\b(medium|large|small)\b/i.test(line) ||
    /runtime_impact.*\b(medium|large|small)\b/i.test(line)
  );
}

const INVALID_TIER_LABELS = [...CANONICAL_CONTRACT.invalidTierLabels];

async function checkSchemaCompile(
  _root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  for (const name of CORE_SCHEMAS) {
    try {
      const schema = await loadSchema(name);
      if (!schema || Object.keys(schema).length === 0) {
        notes.push(`schema ${name} is empty/stub`);
        ok = false;
        continue;
      }
      compileSchema(schema);
      notes.push(`schema ${name} compiles`);
    } catch (err) {
      notes.push(
        `schema ${name} compile error: ${err instanceof Error ? err.message : String(err)}`
      );
      ok = false;
    }
  }
  return { ok, notes };
}

async function checkPolicyKeys(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  try {
    const policy = (await readYamlOrJson(
      path.join(root, "policies", "admission.yaml")
    )) as Record<string, unknown>;
    const requiredKeys = [
      "candidate_completion",
      "success_requires",
      "reject_success_if",
      "outcome_mapping",
    ];
    let ok = true;
    for (const key of requiredKeys) {
      if (key in policy) {
        notes.push(`admission.yaml has ${key}`);
      } else {
        notes.push(`admission.yaml missing ${key}`);
        ok = false;
      }
    }
    return { ok, notes };
  } catch (err) {
    notes.push(
      `admission.yaml read error: ${err instanceof Error ? err.message : String(err)}`
    );
    return { ok: false, notes };
  }
}

async function checkNoPythonCore(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const cliSrc = path.join(root, "packages", "cli", "src");
  let pythonFiles = 0;
  if (await fs.pathExists(cliSrc)) {
    const walk = async (dir: string) => {
      const entries = await fs.readdir(dir, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(dir, entry.name);
        if (entry.isDirectory()) {
          await walk(fullPath);
        } else if (entry.name.endsWith(".py")) {
          pythonFiles++;
        }
      }
    };
    await walk(cliSrc);
  }
  const ok = pythonFiles === 0;
  return {
    ok,
    notes: [
      ok
        ? "no Python files in packages/cli/src"
        : `python files in packages/cli/src (${pythonFiles} found)`,
    ],
  };
}

async function checkPgvWording(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  const files = await walkDoctorFiles(root, DOCTOR_PGV_SCAN_DIRS);

  for (const filePath of files) {
    const content = await fs.readFile(filePath, "utf-8");
    for (const phrase of DANGEROUS_PGV_PHRASES) {
      if (content.includes(phrase)) {
        notes.push(
          `dangerous PGV wording in ${path.relative(root, filePath)}: "${phrase}"`
        );
        ok = false;
      }
    }
  }
  if (ok) {
    notes.push("no dangerous PGV authority wording found");
  }
  return { ok, notes };
}

async function checkTierLabels(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  const excludedFiles = [
    path.join(root, "docs", "RUNTIME_CONTRACT.md"),
    path.join(root, "packages", "cli", "src", "commands", "doctor.ts"),
    path.join(root, "docs", "METRICS.md"),
    path.join(root, "packages", "cli", "src", "core", "metrics.ts"),
    path.join(root, "packages", "cli", "src", "core", "context.ts"),
    path.join(root, "packages", "cli", "src", "core", "contract.ts"),
    path.join(root, "packages", "cli", "src", "core", "recovery.ts"),
    path.join(root, "packages", "cli", "src", "core", "attribution.ts"),
  ];
  const files = await walkDoctorFiles(root, DOCTOR_TIER_SCAN_DIRS);

  for (const fullPath of files) {
    if (excludedFiles.includes(fullPath)) continue;
    const content = await fs.readFile(fullPath, "utf-8");
    const lines = content.split("\n");
    for (const line of lines) {
      if (isAllowedInvalidTierReference(line)) continue;
      for (const label of INVALID_TIER_LABELS) {
        const regex = new RegExp(`\\b${label}\\b`, "i");
        if (regex.test(line)) {
          const rel = path.relative(root, fullPath);
          notes.push(`invalid tier label "${label}" in ${rel}`);
          ok = false;
        }
      }
    }
  }
  if (ok) {
    notes.push("no invalid tier labels found (small/medium/large)");
  }
  return { ok, notes };
}

async function checkAgentsSize(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const agentsPath = path.join(root, "AGENTS.md");
  if (!(await fs.pathExists(agentsPath))) {
    return { ok: false, notes: ["AGENTS.md not found"] };
  }
  const content = await fs.readFile(agentsPath, "utf-8");
  const lines = content.split(/\r?\n/);
  const ok = lines.length <= 150;
  return {
    ok,
    notes: [
      `AGENTS.md is ${lines.length} lines ${ok ? "(<= 150)" : "(> 150)"}`,
    ],
  };
}

async function checkContextFreshness(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const agentsPath = path.join(root, "AGENTS.md");
  if (!(await fs.pathExists(agentsPath))) {
    return { ok: false, notes: ["AGENTS.md not found"] };
  }
  const content = await fs.readFile(agentsPath, "utf-8");
  const result = validateManagedBlock(content);
  return {
    ok: result.valid,
    notes: [result.note],
  };
}

async function checkManagedContractBlocks(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  let contract: Awaited<ReturnType<typeof loadRuntimeContract>>;
  try {
    contract = await loadRuntimeContract(root);
  } catch (err) {
    return {
      ok: false,
      notes: [
        `managed contract load error: ${err instanceof Error ? err.message : String(err)}`,
      ],
    };
  }
  for (const target of MANAGED_CONTRACT_TARGETS) {
    const targetPath = path.join(root, target.path);
    if (!(await fs.pathExists(targetPath))) {
      notes.push(`${target.path} missing`);
      ok = false;
      continue;
    }
    const content = await fs.readFile(targetPath, "utf-8");
    const validation = validateManagedContractBlock(content, target, contract);
    notes.push(validation.note);
    if (!validation.valid) ok = false;
  }
  return { ok, notes };
}

async function checkAdapters(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  for (const adapter of ADAPTERS) {
    const adapterPath = path.join(root, adapter);
    if (await fs.pathExists(adapterPath)) {
      notes.push(`adapter present: ${adapter}`);
    } else {
      notes.push(`adapter missing: ${adapter}`);
      ok = false;
    }
  }
  return { ok, notes };
}

async function checkLocalMarkdownLinks(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  const mdFiles: string[] = [];

  const collectMd = async (dir: string) => {
    const entries = await fs.readdir(dir, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        await collectMd(fullPath);
      } else if (entry.name.endsWith(".md")) {
        mdFiles.push(fullPath);
      }
    }
  };

  if (await fs.pathExists(path.join(root, "docs")))
    await collectMd(path.join(root, "docs"));
  if (await fs.pathExists(path.join(root, "templates")))
    await collectMd(path.join(root, "templates"));

  const linkRegex = /\[([^\]]+)\]\(([^)]+)\)/g;
  for (const mdPath of mdFiles) {
    const content = await fs.readFile(mdPath, "utf-8");
    let match: RegExpExecArray | null;
    while ((match = linkRegex.exec(content)) !== null) {
      const href = match[2];
      if (href.startsWith("http://") || href.startsWith("https://")) continue;
      if (href.startsWith("#")) continue;
      const resolved = path.resolve(path.dirname(mdPath), href);
      if (!(await fs.pathExists(resolved))) {
        const rel = path.relative(root, mdPath);
        notes.push(`broken local link in ${rel}: ${href}`);
        ok = false;
      }
    }
  }
  if (ok) {
    notes.push("no broken local markdown links found");
  }
  return { ok, notes };
}

async function checkEvidenceScopeSupport(
  _root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  try {
    const schema = await loadSchema("completion-card");
    const props = schema.properties as Record<string, unknown> | undefined;
    const evidenceProps = props?.evidence as
      | Record<string, unknown>
      | undefined;
    if (
      evidenceProps?.properties &&
      typeof evidenceProps.properties === "object"
    ) {
      const ep = evidenceProps.properties as Record<string, unknown>;
      if ("verification_artifacts" in ep) {
        notes.push("schema supports verification_artifacts");
      } else {
        notes.push("schema missing verification_artifacts");
        ok = false;
      }
      if ("untested_regions" in ep) {
        notes.push("schema supports untested_regions");
      } else {
        notes.push("schema missing untested_regions");
        ok = false;
      }
      if ("remaining_risks" in ep) {
        notes.push("schema supports remaining_risks");
      } else {
        notes.push("schema missing remaining_risks");
        ok = false;
      }
    } else {
      notes.push("schema evidence block missing properties");
      ok = false;
    }
  } catch (err) {
    notes.push(
      `schema load error: ${err instanceof Error ? err.message : String(err)}`
    );
    ok = false;
  }
  return { ok, notes };
}

async function checkReadOnlyVerifier(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  const verifyGatePath = path.join(root, "docs", "VERIFY_GATE.md");
  if (await fs.pathExists(verifyGatePath)) {
    const content = await fs.readFile(verifyGatePath, "utf-8");
    if (content.includes("read-only")) {
      notes.push("VERIFY_GATE.md states verifier is read-only");
    } else {
      notes.push("VERIFY_GATE.md missing read-only verifier statement");
      ok = false;
    }
  } else {
    notes.push("VERIFY_GATE.md not found");
    ok = false;
  }

  const agentsPath = path.join(root, "AGENTS.md");
  if (await fs.pathExists(agentsPath)) {
    const content = await fs.readFile(agentsPath, "utf-8");
    if (content.includes("read-only")) {
      notes.push("AGENTS.md states verifier is read-only");
    } else {
      notes.push("AGENTS.md missing read-only verifier statement");
      ok = false;
    }
  }
  return { ok, notes };
}

async function checkNoHeavyRuntime(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  const docsDir = path.join(root, "docs");
  if (await fs.pathExists(docsDir)) {
    const entries = await fs.readdir(docsDir, { withFileTypes: true });
    for (const entry of entries) {
      if (!entry.isFile() || !entry.name.endsWith(".md")) continue;
      const content = await fs.readFile(
        path.join(docsDir, entry.name),
        "utf-8"
      );
      const lower = content.toLowerCase();
      if (lower.includes("mandatory mcp") || lower.includes("required mcp")) {
        notes.push(
          `dangerous runtime wording in ${entry.name}: mandatory/required MCP`
        );
        ok = false;
      }
    }
  }
  const readmePath = path.join(root, "README.md");
  if (await fs.pathExists(readmePath)) {
    const content = await fs.readFile(readmePath, "utf-8");
    const lower = content.toLowerCase();
    if (lower.includes("mandatory mcp") || lower.includes("required mcp")) {
      notes.push("README.md contains mandatory/required MCP wording");
      ok = false;
    }
  }
  if (ok) {
    notes.push("no mandatory MCP/required heavy runtime wording found");
  }
  return { ok, notes };
}

async function checkTemplatesInventory(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  const templatesDir = path.join(root, "templates");
  if (!(await fs.pathExists(templatesDir))) {
    notes.push("templates directory missing");
    return { ok: false, notes };
  }

  const entries = await fs.readdir(templatesDir, { withFileTypes: true });
  const mdFiles = entries
    .filter((e) => e.isFile() && e.name.endsWith(".md"))
    .map((e) => e.name)
    .sort();

  if (mdFiles.length === 0) {
    notes.push("no markdown templates found");
    ok = false;
  } else {
    notes.push(`${mdFiles.length} template(s) present: ${mdFiles.join(", ")}`);
  }

  // Ensure core tier templates exist
  const coreTemplates = [
    "SUBAGENT_TASK_light.md",
    "SUBAGENT_TASK_standard.md",
    "SUBAGENT_TASK_deep.md",
    "COMPLETION_CARD.md",
  ];
  for (const core of coreTemplates) {
    if (!mdFiles.includes(core)) {
      notes.push(`core template missing: ${core}`);
      ok = false;
    }
  }

  return { ok, notes };
}

async function checkPolicyDrift(
  root: string
): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  const policyPath = path.join(root, "policies", "admission.yaml");

  let policy: Record<string, unknown>;
  try {
    policy = (await readYamlOrJson(policyPath)) as Record<string, unknown>;
  } catch (err) {
    notes.push(
      `admission.yaml read error: ${err instanceof Error ? err.message : String(err)}`
    );
    return { ok: false, notes };
  }

  // 1. Validate required policy sections exist
  const requiredSections = [
    "candidate_completion",
    "success_requires",
    "reject_success_if",
    "outcome_mapping",
    "evidence_floor",
  ];
  for (const section of requiredSections) {
    if (section in policy) {
      notes.push(`policy section ${section} present`);
    } else {
      notes.push(`policy section ${section} missing`);
      ok = false;
    }
  }

  // 2. Validate candidate_completion.required fields map to schema known fields
  const candidateCompletion = policy.candidate_completion as
    | Record<string, unknown>
    | undefined;
  if (candidateCompletion && Array.isArray(candidateCompletion.required)) {
    const requiredFields = candidateCompletion.required as string[];
    for (const field of requiredFields) {
      // Normalize dotted paths like "claim.fix_status" to just check top-level "claim"
      const topLevel = field.split(".")[0];
      if (KNOWN_SCHEMA_REQUIRED_FIELDS.includes(topLevel)) {
        notes.push(`candidate_completion.required field known: ${field}`);
      } else {
        notes.push(`candidate_completion.required field unknown: ${field}`);
        ok = false;
      }
    }
  } else {
    notes.push("candidate_completion.required missing or not an array");
    ok = false;
  }

  // 3. Validate success_requires predicates are known
  const successRequires = policy.success_requires as string[] | undefined;
  if (successRequires && Array.isArray(successRequires)) {
    for (const predicate of successRequires) {
      const normalized = predicate.trim();
      const isKnown = KNOWN_SUCCESS_PREDICATES.some((kp) =>
        normalized.toLowerCase().startsWith(kp.toLowerCase())
      );
      if (isKnown) {
        notes.push(`success_requires predicate known: ${predicate}`);
      } else {
        notes.push(`success_requires predicate unknown: ${predicate}`);
        ok = false;
      }
    }
  } else {
    notes.push("success_requires missing or not an array");
    ok = false;
  }

  // 4. Validate evidence_floor tier labels map to schema evidence properties and admission.ts expectations
  const evidenceFloor = policy.evidence_floor as
    | Record<string, unknown>
    | undefined;
  if (evidenceFloor && typeof evidenceFloor === "object") {
    for (const tier of ["light", "standard", "deep"] as const) {
      const tierConfig = evidenceFloor[tier] as
        | Record<string, unknown>
        | undefined;
      if (!tierConfig) {
        notes.push(`evidence_floor missing tier: ${tier}`);
        ok = false;
        continue;
      }

      const knownLabels = KNOWN_TIER_EVIDENCE_LABELS[tier];
      const canonicalTier =
        CANONICAL_CONTRACT.evidenceFloor[
          tier as keyof typeof CANONICAL_CONTRACT.evidenceFloor
        ];
      for (const key of ["required", "one_of", "recommended"] as const) {
        const labels = tierConfig[key] as string[] | undefined;
        if (!labels) continue;
        if (!Array.isArray(labels)) {
          notes.push(`evidence_floor.${tier}.${key} is not an array`);
          ok = false;
          continue;
        }
        for (const label of labels) {
          if (knownLabels.includes(label)) {
            notes.push(`evidence_floor.${tier} label known: ${label}`);
          } else {
            notes.push(`evidence_floor.${tier} label unknown: ${label}`);
            ok = false;
          }
        }
      }

      for (const key of [
        ["required", canonicalTier.required],
        ["one_of", "oneOf" in canonicalTier ? canonicalTier.oneOf : []],
        [
          "runtime_enforced",
          "runtimeEnforced" in canonicalTier
            ? canonicalTier.runtimeEnforced
            : [],
        ],
      ] as const) {
        const [policyKey, expectedLabels] = key;
        const labels = tierConfig[policyKey] as string[] | undefined;
        if (!expectedLabels || expectedLabels.length === 0) continue;
        if (!Array.isArray(labels)) {
          notes.push(`evidence_floor.${tier}.${policyKey} missing`);
          ok = false;
          continue;
        }
        for (const expectedLabel of expectedLabels) {
          if (labels.includes(expectedLabel)) {
            notes.push(
              `evidence_floor.${tier}.${policyKey} includes ${expectedLabel}`
            );
          } else {
            notes.push(
              `evidence_floor.${tier}.${policyKey} missing ${expectedLabel}`
            );
            ok = false;
          }
        }
      }
    }
  } else {
    notes.push("evidence_floor missing or not an object");
    ok = false;
  }

  // 5. Validate outcome_mapping keys are known outcomes
  const outcomeMapping = policy.outcome_mapping as
    | Record<string, unknown>
    | undefined;
  if (outcomeMapping && typeof outcomeMapping === "object") {
    for (const key of Object.keys(outcomeMapping)) {
      if (KNOWN_OUTCOMES.includes(key)) {
        notes.push(`outcome_mapping key known: ${key}`);
      } else {
        notes.push(`outcome_mapping key unknown: ${key}`);
        ok = false;
      }
    }
  } else {
    notes.push("outcome_mapping missing or not an object");
    ok = false;
  }

  // 6. Validate reject_success_if keys are known rejection fields
  const rejectSuccessIf = policy.reject_success_if as
    | Record<string, unknown>
    | undefined;
  if (rejectSuccessIf && typeof rejectSuccessIf === "object") {
    for (const key of Object.keys(rejectSuccessIf)) {
      if (KNOWN_REJECT_KEYS.includes(key)) {
        notes.push(`reject_success_if key known: ${key}`);
      } else {
        notes.push(`reject_success_if key unknown: ${key}`);
        ok = false;
      }
    }
  } else {
    notes.push("reject_success_if missing or not an object");
    ok = false;
  }

  return { ok, notes };
}

export function doctorCommand(): Command {
  return new Command("doctor")
    .description(
      "Check required files, schemas, policies, templates, and adapters"
    )
    .option("--root <path>", "Repository root", process.cwd())
    .option("--policy-drift", "Run policy-code drift checks", false)
    .action(async (opts: { root: string; policyDrift: boolean }) => {
      const root = path.resolve(opts.root);
      const missing: string[] = [];
      const present: string[] = [];
      const notes: string[] = [];
      const checks: { name: string; status: "pass" | "fail"; note: string }[] =
        [];

      // Required file check
      for (const asset of CRITICAL_ASSETS) {
        const assetPath = path.join(root, asset);
        if (await fs.pathExists(assetPath)) {
          present.push(asset);
        } else {
          missing.push(asset);
        }
      }
      checks.push({
        name: "required_files",
        status: missing.length === 0 ? "pass" : "fail",
        note:
          missing.length === 0
            ? "all required files present"
            : `missing: ${missing.join(", ")}`,
      });

      // Schema compile check
      const schemaResult = await checkSchemaCompile(root);
      checks.push({
        name: "schema_compile",
        status: schemaResult.ok ? "pass" : "fail",
        note: schemaResult.notes.join("; "),
      });

      // Policy key check
      const policyResult = await checkPolicyKeys(root);
      checks.push({
        name: "policy_keys",
        status: policyResult.ok ? "pass" : "fail",
        note: policyResult.notes.join("; "),
      });

      // Policy-code drift check (always run; --policy-drift makes the request explicit)
      const driftResult = await checkPolicyDrift(root);
      const driftNote = opts.policyDrift
        ? `[explicit] ${driftResult.notes.join("; ")}`
        : driftResult.notes.join("; ");
      checks.push({
        name: "policy_drift",
        status: driftResult.ok ? "pass" : "fail",
        note: driftNote,
      });

      // No Python core check
      const pythonResult = await checkNoPythonCore(root);
      checks.push({
        name: "no_python_core",
        status: pythonResult.ok ? "pass" : "fail",
        note: pythonResult.notes.join("; "),
      });

      // PGV authority wording check
      const pgvResult = await checkPgvWording(root);
      checks.push({
        name: "pgv_authority_wording",
        status: pgvResult.ok ? "pass" : "fail",
        note: pgvResult.notes.join("; "),
      });

      // Tier label check
      const tierResult = await checkTierLabels(root);
      checks.push({
        name: "tier_labels",
        status: tierResult.ok ? "pass" : "fail",
        note: tierResult.notes.join("; "),
      });

      // AGENTS size check
      const agentsResult = await checkAgentsSize(root);
      checks.push({
        name: "agents_size",
        status: agentsResult.ok ? "pass" : "fail",
        note: agentsResult.notes.join("; "),
      });

      // Adapter presence check
      const adapterResult = await checkAdapters(root);
      checks.push({
        name: "adapters_present",
        status: adapterResult.ok ? "pass" : "fail",
        note: adapterResult.notes.join("; "),
      });

      // Local markdown link check
      const linkResult = await checkLocalMarkdownLinks(root);
      checks.push({
        name: "local_markdown_links",
        status: linkResult.ok ? "pass" : "fail",
        note: linkResult.notes.join("; "),
      });

      // Evidence scope support check
      const evidenceScopeResult = await checkEvidenceScopeSupport(root);
      checks.push({
        name: "evidence_scope_support",
        status: evidenceScopeResult.ok ? "pass" : "fail",
        note: evidenceScopeResult.notes.join("; "),
      });

      // Component registry check
      const componentRegistryResult = await validateComponentsRegistry(root);
      checks.push({
        name: "component_registry",
        status: componentRegistryResult.ok ? "pass" : "fail",
        note: componentRegistryResult.ok
          ? `${componentRegistryResult.component_count} component(s); protected paths ${componentRegistryResult.protected_paths_covered}/${componentRegistryResult.protected_paths_checked} covered`
          : componentRegistryResult.errors.join("; "),
      });

      // Read-only verifier check
      const readOnlyResult = await checkReadOnlyVerifier(root);
      checks.push({
        name: "read_only_verifier",
        status: readOnlyResult.ok ? "pass" : "fail",
        note: readOnlyResult.notes.join("; "),
      });

      // No heavy runtime check
      const runtimeResult = await checkNoHeavyRuntime(root);
      checks.push({
        name: "no_heavy_runtime",
        status: runtimeResult.ok ? "pass" : "fail",
        note: runtimeResult.notes.join("; "),
      });

      // Templates inventory check
      const templatesResult = await checkTemplatesInventory(root);
      checks.push({
        name: "templates_inventory",
        status: templatesResult.ok ? "pass" : "fail",
        note: templatesResult.notes.join("; "),
      });

      // Cleanup policy check (advisory-only)
      const cleanupPath = path.join(root, "policies", "cleanup.yaml");
      if (await fs.pathExists(cleanupPath)) {
        checks.push({
          name: "cleanup_policy",
          status: "pass",
          note: "cleanup policy present",
        });
      } else {
        checks.push({
          name: "cleanup_policy",
          status: "pass",
          note: "cleanup policy optional; not required for v0.1",
        });
      }

      // Context freshness check
      const contextFreshnessResult = await checkContextFreshness(root);
      checks.push({
        name: "context_freshness",
        status: contextFreshnessResult.ok ? "pass" : "fail",
        note: contextFreshnessResult.notes.join("; "),
      });

      // Managed runtime contract blocks in docs/templates/adapters
      const managedContractResult = await checkManagedContractBlocks(root);
      checks.push({
        name: "managed_contract_blocks",
        status: managedContractResult.ok ? "pass" : "fail",
        note: managedContractResult.notes.join("; "),
      });

      const healthy = checks.every((c) => c.status === "pass");

      const report = {
        healthy,
        present_count: present.length,
        missing_count: missing.length,
        present,
        missing,
        checks,
        notes,
      };

      console.log(JSON.stringify(report, null, 2));
      process.exit(healthy ? 0 : 1);
    });
}
