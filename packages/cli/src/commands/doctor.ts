import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import { loadSchema, compileSchema, readYamlOrJson } from "../core/schema.js";

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
  "templates/COMPLETION_CARD.md",
  "templates/SUBAGENT_TASK_light.md",
  "templates/SUBAGENT_TASK_standard.md",
  "templates/SUBAGENT_TASK_deep.md",
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

const VALID_TIER_LABELS = ["light", "standard", "deep"];
const INVALID_TIER_LABELS = ["small", "medium", "large"];

async function checkSchemaCompile(root: string): Promise<{ ok: boolean; notes: string[] }> {
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
      notes.push(`schema ${name} compile error: ${err instanceof Error ? err.message : String(err)}`);
      ok = false;
    }
  }
  return { ok, notes };
}

async function checkPolicyKeys(root: string): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  try {
    const policy = await readYamlOrJson(path.join(root, "policies", "admission.yaml")) as Record<string, unknown>;
    const requiredKeys = ["candidate_completion", "success_requires", "reject_success_if", "outcome_mapping"];
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
    notes.push(`admission.yaml read error: ${err instanceof Error ? err.message : String(err)}`);
    return { ok: false, notes };
  }
}

async function checkNoPythonCore(root: string): Promise<{ ok: boolean; notes: string[] }> {
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
  return { ok, notes: [ok ? "no Python files in packages/cli/src" : `python files in packages/cli/src (${pythonFiles} found)`] };
}

async function checkPgvWording(root: string): Promise<{ ok: boolean; notes: string[] }> {
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

async function checkTierLabels(root: string): Promise<{ ok: boolean; notes: string[] }> {
  const notes: string[] = [];
  let ok = true;
  const excludedFiles = [
    path.join(root, "docs", "RUNTIME_CONTRACT.md"),
    path.join(root, "packages", "cli", "src", "commands", "doctor.ts"),
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

async function checkAgentsSize(root: string): Promise<{ ok: boolean; notes: string[] }> {
  const agentsPath = path.join(root, "AGENTS.md");
  if (!(await fs.pathExists(agentsPath))) {
    return { ok: false, notes: ["AGENTS.md not found"] };
  }
  const content = await fs.readFile(agentsPath, "utf-8");
  const lines = content.split(/\r?\n/);
  const ok = lines.length <= 150;
  return { ok, notes: [`AGENTS.md is ${lines.length} lines ${ok ? "(<= 150)" : "(> 150)"}`] };
}

async function checkAdapters(root: string): Promise<{ ok: boolean; notes: string[] }> {
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

async function checkLocalMarkdownLinks(root: string): Promise<{ ok: boolean; notes: string[] }> {
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

  if (await fs.pathExists(path.join(root, "docs"))) await collectMd(path.join(root, "docs"));
  if (await fs.pathExists(path.join(root, "templates"))) await collectMd(path.join(root, "templates"));

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

export function doctorCommand(): Command {
  return new Command("doctor")
    .description("Check required files, schemas, policies, templates, and adapters")
    .option("--root <path>", "Repository root", process.cwd())
    .action(async (opts: { root: string }) => {
      const root = path.resolve(opts.root);
      const missing: string[] = [];
      const present: string[] = [];
      const notes: string[] = [];
      const checks: { name: string; status: "pass" | "fail"; note: string }[] = [];

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
        note: missing.length === 0 ? "all required files present" : `missing: ${missing.join(", ")}`,
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
