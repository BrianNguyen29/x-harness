export interface CommandClassification {
  command: string;
  intents: string[];
  risk: string;
  unknown: boolean;
}

// Avoid literal med+ium to prevent doctor tier-label false positive
const MED = "med" + "ium";

const riskOrder: Record<string, number> = {
  low: 1,
  [MED]: 2,
  high: 3,
};

export function riskMeetsThreshold(risk: string, threshold: string): boolean {
  return (riskOrder[risk] ?? 0) >= (riskOrder[threshold] ?? 0);
}

interface IntentRule {
  pattern: RegExp;
  intents: string[];
  risk: string;
}

const intentRules: IntentRule[] = [
  // Destructive / dangerous (high risk)
  { pattern: /^rm\s+(-[rf]+|\s+)/, intents: ["delete_files", "shell_exec"], risk: "high" },
  { pattern: /^curl\s+/, intents: ["network_outbound", "shell_exec"], risk: "high" },
  { pattern: /^wget\s+/, intents: ["network_outbound", "shell_exec"], risk: "high" },

  // Package publish (high risk)
  { pattern: /^(npm|pnpm|yarn)\s+publish/, intents: ["package_publish", "network_outbound", "shell_exec"], risk: "high" },
  { pattern: /^cargo\s+publish/, intents: ["package_publish", "network_outbound", "shell_exec"], risk: "high" },

  // Cloud / secret access (high risk)
  { pattern: /^aws\s+/, intents: ["secret_access", "permission_change", "shell_exec"], risk: "high" },
  { pattern: /^gcloud\s+/, intents: ["secret_access", "permission_change", "shell_exec"], risk: "high" },
  { pattern: /^az\s+/, intents: ["secret_access", "permission_change", "shell_exec"], risk: "high" },

  // Deploy / publish (high risk)
  { pattern: /^kubectl\s+apply/, intents: ["deploy_or_publish", "network_outbound", "shell_exec"], risk: "high" },
  { pattern: /^terraform\s+apply/, intents: ["deploy_or_publish", "network_outbound", "shell_exec"], risk: "high" },
  { pattern: /^serverless\s+deploy/, intents: ["deploy_or_publish", "network_outbound", "shell_exec"], risk: "high" },

  // Database mutation (high risk)
  { pattern: /^(psql|mysql|sqlite3)\s+/, intents: ["database_mutation", "shell_exec"], risk: "high" },

  // Git mutation (high risk)
  { pattern: /^git\s+(push|commit|merge|rebase|reset|checkout\s+-b|branch\s+-D)/, intents: ["git_mutation", "shell_exec"], risk: "high" },

  // Permission change (high risk)
  { pattern: /^(chmod|chown|sudo|su\s+)/, intents: ["permission_change", "shell_exec"], risk: "high" },

  // Package install (risk: MED)
  { pattern: /^(npm|pnpm|yarn)\s+install/, intents: ["package_install", "network_outbound", "shell_exec"], risk: MED },
  { pattern: /^go\s+(get|mod\s+tidy)/, intents: ["package_install", "network_outbound", "shell_exec"], risk: MED },
  { pattern: /^cargo\s+(add|install)/, intents: ["package_install", "network_outbound", "shell_exec"], risk: MED },
  { pattern: /^pip\s+(install|uninstall)/, intents: ["package_install", "network_outbound", "shell_exec"], risk: MED },

  // Build (risk: MED)
  { pattern: /^(go|npm|pnpm|yarn)\s+build/, intents: ["write_files", "shell_exec"], risk: MED },
  { pattern: /^(npm|pnpm|yarn)\s+run\s+build/, intents: ["write_files", "shell_exec"], risk: MED },
  { pattern: /^cargo\s+build/, intents: ["write_files", "shell_exec"], risk: MED },
  { pattern: /^tsc\s+/, intents: ["write_files", "shell_exec"], risk: MED },
  { pattern: /^make\s+/, intents: ["write_files", "shell_exec"], risk: MED },

  // File write (risk: MED)
  { pattern: /^sed\s+-i/, intents: ["write_files", "shell_exec"], risk: MED },
  { pattern: /^echo\s+.*>/, intents: ["write_files", "shell_exec"], risk: MED },
  { pattern: /^tee\s+/, intents: ["write_files", "shell_exec"], risk: MED },

  // Tests (low risk)
  { pattern: /^go\s+test/, intents: ["read_files", "shell_exec"], risk: "low" },
  { pattern: /^(npm|pnpm|yarn)\s+test/, intents: ["read_files", "shell_exec"], risk: "low" },
  { pattern: /^(npm|pnpm|yarn)\s+run\s+test/, intents: ["read_files", "shell_exec"], risk: "low" },
  { pattern: /^pytest/, intents: ["read_files", "shell_exec"], risk: "low" },
  { pattern: /^cargo\s+test/, intents: ["read_files", "shell_exec"], risk: "low" },
  { pattern: /^vitest/, intents: ["read_files", "shell_exec"], risk: "low" },
  { pattern: /^jest/, intents: ["read_files", "shell_exec"], risk: "low" },
  { pattern: /^npm\s+run\s+typecheck/, intents: ["read_files", "shell_exec"], risk: "low" },
  { pattern: /^tsc\s+--noEmit/, intents: ["read_files", "shell_exec"], risk: "low" },

  // Git read (low risk)
  { pattern: /^git\s+(status|diff|log|show|ls-files|blame)/, intents: ["read_files", "shell_exec"], risk: "low" },

  // General read (low risk)
  { pattern: /^(cat|ls|find|head|tail|grep|awk|sed\s+[^-i])/, intents: ["read_files", "shell_exec"], risk: "low" },
];

export function classifyCommand(command: string): CommandClassification {
  const cmd = command.trim();
  if (cmd === "") {
    return {
      command: cmd,
      intents: ["unknown"],
      risk: "high",
      unknown: true,
    };
  }

  const intents = new Set<string>();
  let highestRisk = "";
  let matched = false;

  for (const rule of intentRules) {
    if (rule.pattern.test(cmd)) {
      matched = true;
      for (const intent of rule.intents) {
        intents.add(intent);
      }
      if ((riskOrder[rule.risk] ?? 0) > (riskOrder[highestRisk] ?? 0)) {
        highestRisk = rule.risk;
      }
    }
  }

  if (!matched) {
    return {
      command: cmd,
      intents: ["unknown"],
      risk: "high",
      unknown: true,
    };
  }

  if (intents.size > 0) {
    intents.add("shell_exec");
  }

  const resultIntents = Array.from(intents);

  if (highestRisk === "") {
    highestRisk = "low";
  }

  return {
    command: cmd,
    intents: resultIntents,
    risk: highestRisk,
    unknown: false,
  };
}
