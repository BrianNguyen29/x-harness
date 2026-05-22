import { createHash } from "node:crypto";

const CANONICAL_RULES = [
  "Completion is admitted, not claimed.",
  "Verifier is read-only.",
  "Success is the only accepted outcome.",
  "Canonical tiers: light, standard, deep.",
  "PGV is advisory-only.",
];

const MANAGED_BEGIN = "<!-- BEGIN X-HARNESS MANAGED CONTEXT -->";
const MANAGED_END = "<!-- END X-HARNESS MANAGED CONTEXT -->";

export function getCanonicalContext(verbose = false): string {
  if (!verbose) {
    return CANONICAL_RULES.join("\n");
  }
  return [
    "# x-harness Canonical Context",
    "",
    ...CANONICAL_RULES.map((r) => `- ${r}`),
    "",
    "## Rules",
    "",
    "### Completion is admitted, not claimed",
    "Agents may propose completion but cannot self-admit. A result with `fix_status: fixed` is only a completion candidate.",
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

export function getContextHash(text: string): string {
  return createHash("sha256").update(text, "utf-8").digest("hex").slice(0, 16);
}

export function generateManagedBlock(): string {
  const context = getCanonicalContext(true);
  const hash = getContextHash(context);
  const generatedAt = new Date().toISOString();
  return [
    MANAGED_BEGIN,
    "<!-- generated-by: x-harness -->",
    `<!-- generated-at: ${generatedAt} -->`,
    `<!-- context-hash: ${hash} -->`,
    "",
    context,
    "",
    MANAGED_END,
  ].join("\n");
}

export function injectManagedBlock(
  agentsContent: string,
  block: string
): string {
  const beginIndex = agentsContent.indexOf(MANAGED_BEGIN);
  const endIndex = agentsContent.indexOf(MANAGED_END);

  if (beginIndex !== -1 && endIndex !== -1 && endIndex > beginIndex) {
    // Replace existing block
    const before = agentsContent.slice(0, beginIndex);
    const after = agentsContent.slice(endIndex + MANAGED_END.length);
    return before + block + after;
  }

  // Append block at the end
  const separator = agentsContent.endsWith("\n") ? "" : "\n\n";
  return agentsContent + separator + block + "\n";
}

export function extractManagedBlock(agentsContent: string): string | null {
  const beginIndex = agentsContent.indexOf(MANAGED_BEGIN);
  const endIndex = agentsContent.indexOf(MANAGED_END);
  if (beginIndex === -1 || endIndex === -1 || endIndex < beginIndex) {
    return null;
  }
  return agentsContent.slice(beginIndex, endIndex + MANAGED_END.length);
}

export function validateManagedBlock(agentsContent: string): {
  valid: boolean;
  note: string;
} {
  const block = extractManagedBlock(agentsContent);
  if (!block) {
    return {
      valid: false,
      note: "AGENTS.md missing managed context block",
    };
  }

  const hashMatch = block.match(/<!-- context-hash: ([a-f0-9]+) -->/);
  if (!hashMatch) {
    return {
      valid: false,
      note: "AGENTS.md managed block missing context-hash",
    };
  }

  const currentContext = getCanonicalContext(true);
  const expectedHash = getContextHash(currentContext);
  const actualHash = hashMatch[1];

  if (actualHash !== expectedHash) {
    return {
      valid: false,
      note: `AGENTS.md context hash stale: expected ${expectedHash}, found ${actualHash}`,
    };
  }

  return {
    valid: true,
    note: "AGENTS.md managed context block is fresh",
  };
}

export function getCompactContextHeader(): string {
  const lines = [
    "## Context",
    "",
    ...CANONICAL_RULES.map((r) => `- ${r}`),
    "",
    "For full context run: `node packages/cli/dist/index.js context`",
    "",
  ];
  return lines.join("\n");
}
