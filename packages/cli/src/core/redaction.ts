export type RedactionMode = "secret-redaction";

export interface RedactionFinding {
  pattern: string;
  replacements: number;
}

export interface RedactionResult {
  text: string;
  findings: RedactionFinding[];
  replacements: number;
}

interface RedactionPattern {
  id: string;
  regex: RegExp;
  replace: string | ((match: string, ...groups: string[]) => string);
}

const REDACTION_PATTERNS: RedactionPattern[] = [
  {
    id: "private_key",
    regex:
      /-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----/g,
    replace: "[REDACTED:private_key]",
  },
  {
    id: "github_token",
    regex: /\bgh[pousr]_[A-Za-z0-9_]{20,}\b/g,
    replace: "[REDACTED:github_token]",
  },
  {
    id: "npm_token",
    regex: /\bnpm_[A-Za-z0-9]{20,}\b/g,
    replace: "[REDACTED:npm_token]",
  },
  {
    id: "bearer_token",
    regex: /\bBearer\s+([A-Za-z0-9._~+/=-]{10,})\b/g,
    replace: "Bearer [REDACTED:bearer_token]",
  },
  {
    id: "jwt",
    regex: /\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b/g,
    replace: "[REDACTED:jwt]",
  },
  {
    id: "connection_string",
    regex: /\b((?:postgres(?:ql)?|mysql|mongodb|redis):\/\/)[^\s"'<>]+/gi,
    replace: (_match, protocol: string) =>
      `${protocol}[REDACTED:connection_string]`,
  },
  {
    id: "api_key",
    regex:
      /\b(api[_-]?key|apikey|access[_-]?key|secret[_-]?key)\s*[:=]\s*["']?([A-Za-z0-9._~+/=-]{12,})["']?/gi,
    replace: (_match, key: string) => `${key}=[REDACTED:api_key]`,
  },
  {
    id: "password_assignment",
    regex: /\b(password|passwd|pwd)\s*[:=]\s*["']?([^\s"'`]{6,})["']?/gi,
    replace: (_match, key: string) => `${key}=[REDACTED:password_assignment]`,
  },
];

export function redactText(
  input: string,
  _mode: RedactionMode = "secret-redaction"
): RedactionResult {
  let text = input;
  const findings: RedactionFinding[] = [];

  for (const pattern of REDACTION_PATTERNS) {
    let replacements = 0;
    text = text.replace(pattern.regex, (...args: string[]) => {
      replacements += 1;
      if (typeof pattern.replace === "function") {
        return pattern.replace(args[0], ...args.slice(1));
      }
      return pattern.replace;
    });
    if (replacements > 0) {
      findings.push({ pattern: pattern.id, replacements });
    }
  }

  return {
    text,
    findings,
    replacements: findings.reduce(
      (sum, finding) => sum + finding.replacements,
      0
    ),
  };
}

export function redactObject<T>(input: T): {
  value: T;
  result: RedactionResult;
} {
  const raw = JSON.stringify(input);
  const result = redactText(raw);
  return {
    value: JSON.parse(result.text) as T,
    result,
  };
}
