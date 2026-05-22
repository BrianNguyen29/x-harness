import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { loadSchema, compileSchema, readYamlOrJson } from "../core/schema.js";
import { validateManagedBlock } from "../core/context.js";

// Known predicates/fields from the TypeScript admission engine (runtime source of truth)
const KNOWN_SCHEMA_REQUIRED_FIELDS = [
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
];

const KNOWN_SUCCESS_PREDICATES = [
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
];

const KNOWN_TIER_EVIDENCE_LABELS: Record<string, string[]> = {
  light: ["files_changed", "command_evidence", "manual_rationale"],
  standard: [
    "files_changed",
    "command_evidence",
    "evidence_scope_declared",
    "untested_regions_declared",
  ],
  deep: [
    "files_changed",
    "command_evidence",
    "evidence_scope_declared",
    "untested_regions_declared",
    "remaining_risks_declared",
    "execution_controls_present",
    "rollback_policy_present",
  ],
};

const KNOWN_OUTCOMES = [
  "success",
  "failed",
  "blocked",
  "skipped",
  "timeout",
  "error",
];

const KNOWN_REJECT_KEYS = [
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
  "docs/VERIFY_GATE.md",
  "docs/RUNTIME_CONTRACT.md",
  "docs/ADMISSION_POLICY.md",
  "docs/PGV_ADVISORY.md",
  "docs/DENOMINATOR_POLICY.md",
  "docs/ROADMAP.md",
  "docs/ADAPTERS.md",
  "docs/RECOVERY.md",
  "docs/METRICS.md",
  "templates/COMPLETION_CARD.md",
  "templates/SUBAGENT_TASK_light.md",
  "templates/SUBAGENT_TASK_standard.md",
  "templates/SUBAGENT_TASK_deep.md",
  "templates/HARNESS_CHANGE_CONTRACT.md",
  "schemas/completion-card.schema.json",
  "schemas/subagent-return.schema.json",
  "schemas/verify-event.schema.json",
  "schemas/pgv-advice.schema.json",
  "policies/admission.yaml",
];

const CORE_SCHEMAS = [
  "completion-card",
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

const INVALID_TIER_LABELS = ["small", "medium", "large"];

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
  const docsDir = path.join(root, "docs");
  const coreDir = path.join(root, "packages", "cli", "src", "core");
  const dirs = [docsDir, coreDir].filter((d) => fs.pathExistsSync(d));

  for (const dir of dirs) {
    const entries = await fs.readdir(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (!entry.isFile() || !entry.name.endsWith(".md")) continue;
      const content = await fs.readFile(path.join(dir, entry.name), "utf-8");
      for (const phrase of DANGEROUS_PGV_PHRASES) {
        if (content.includes(phrase)) {
          notes.push(`dangerous PGV wording in ${entry.name}: "${phrase}"`);
          ok = false;
        }
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
    path.join(root, "packages", "cli", "src", "core", "recovery.ts"),
  ];
  const dirs = [
    path.join(root, "docs"),
    path.join(root, "packages", "cli", "src"),
  ].filter((d) => fs.pathExistsSync(d));

  for (const dir of dirs) {
    const walk = async (d: string) => {
      const entries = await fs.readdir(d, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(d, entry.name);
        if (entry.isDirectory()) {
          await walk(fullPath);
        } else if (entry.name.endsWith(".md") || entry.name.endsWith(".ts")) {
          if (excludedFiles.includes(fullPath)) continue;
          const content = await fs.readFile(fullPath, "utf-8");
          for (const label of INVALID_TIER_LABELS) {
            const regex = new RegExp(`\\b${label}\\b`, "i");
            if (regex.test(content)) {
              const rel = path.relative(root, fullPath);
              notes.push(`invalid tier label "${label}" in ${rel}`);
              ok = false;
            }
          }
        }
      }
    };
    await walk(dir);
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

      // Policy-code drift check (always run; --policy-drift can be used to surface it explicitly)
      const driftResult = await checkPolicyDrift(root);
      checks.push({
        name: "policy_drift",
        status: driftResult.ok ? "pass" : "fail",
        note: driftResult.notes.join("; "),
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
