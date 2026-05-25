import {
  getContractHash,
  renderCanonicalContext,
  renderCompactContextHeader,
} from "./contract.js";

const MANAGED_BEGIN = "<!-- BEGIN X-HARNESS MANAGED CONTEXT -->";
const MANAGED_END = "<!-- END X-HARNESS MANAGED CONTEXT -->";

export function getCanonicalContext(verbose = false): string {
  return renderCanonicalContext(verbose);
}

export function getContextHash(text: string): string {
  return getContractHash(text);
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
  const actualContext = block
    .split("\n")
    .filter((line) => {
      const trimmed = line.trim();
      return (
        trimmed !== MANAGED_BEGIN &&
        trimmed !== MANAGED_END &&
        !trimmed.startsWith("<!--")
      );
    })
    .join("\n")
    .trim();

  if (actualHash !== expectedHash) {
    return {
      valid: false,
      note: `AGENTS.md context hash stale: expected ${expectedHash}, found ${actualHash}`,
    };
  }

  if (actualContext !== currentContext.trim()) {
    return {
      valid: false,
      note: "AGENTS.md managed context body differs from canonical context",
    };
  }

  return {
    valid: true,
    note: "AGENTS.md managed context block is fresh",
  };
}

export function getCompactContextHeader(): string {
  return renderCompactContextHeader();
}
