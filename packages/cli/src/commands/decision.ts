import { Command } from "commander";
import * as path from "node:path";
import {
  applyDecisionLinkRefs,
  buildDecisionRecord,
  collectDecisionLinkRefs,
  DECISION_RECORD_DEFAULT_DIR,
  DECISION_RECORD_STATUSES,
  defaultDecisionOutputPath,
  ensureDecisionOutputDir,
  formatDecisionLinkList,
  isJsonExtension,
  listDecisionRecords,
  loadCompletionCardForLink,
  matchDecisionAffected,
  matchDecisionQuery,
  normalizeDecisionStatus,
  serializeDecisionRecord,
  writeDecisionRecord,
  type DecisionListEntry,
  type DecisionRecord,
  type DecisionRecordSpec,
} from "../core/decision.js";
import { DECISION_LINK_RESULT_SCHEMA_VERSION } from "../core/decision.js";

type ExitCode = number;

const EXIT_OK = 0;
const EXIT_USAGE = 2;
const EXIT_ERROR = 1;

interface ParsedFlagValue {
  value: string;
  consumed: number;
}

class FlagError extends Error {
  constructor(
    message: string,
    readonly exitCode: ExitCode = EXIT_USAGE
  ) {
    super(message);
    this.name = "FlagError";
  }
}

function readFlag(
  args: string[],
  i: number,
  flag: string,
  _list = false
): ParsedFlagValue {
  if (i + 1 >= args.length) {
    throw new FlagError(`error: ${flag} requires a value`);
  }
  return { value: args[i + 1], consumed: 2 };
}

function appendSplit(target: string[], value: string): void {
  for (const part of value.split(",")) {
    const trimmed = part.trim();
    if (trimmed === "") continue;
    if (!target.includes(trimmed)) {
      target.push(trimmed);
    }
  }
}

interface RecordActionOptions {
  id?: string;
  title?: string;
  date?: string;
  status?: string;
  decision?: string;
  rationale?: string;
  context?: string;
  consequence?: string;
  supersededBy?: string;
  tag?: string[];
  affectedPath?: string[];
  note?: string;
  output?: string;
  json?: boolean;
}

interface ListActionOptions {
  dir?: string;
  json?: boolean;
}

interface QueryActionOptions {
  dir?: string;
  keyword?: string;
  json?: boolean;
}

interface AffectedActionOptions {
  dir?: string;
  path?: string;
  json?: boolean;
}

interface LinkActionOptions {
  card?: string;
  decision?: string[];
  out?: string;
  json?: boolean;
}

function renderRecordText(outPath: string): void {
  console.log(`Decision record written: ${outPath}`);
}

function renderRecordJson(payload: {
  path: string;
  record: DecisionRecord;
}): void {
  console.log(JSON.stringify(payload, null, 2));
}

async function runRecordAction(opts: RecordActionOptions): Promise<number> {
  const spec: DecisionRecordSpec = {
    id: opts.id ?? "",
    title: opts.title ?? "",
    date: opts.date ?? "",
    // Default to the safe-V1 status so the spec round-trips through
    // normalizeDecisionStatus without altering user input.
    status: opts.status ?? "proposed",
    decision: opts.decision ?? "",
    rationale: opts.rationale ?? "",
    context: opts.context ?? "",
    consequences: opts.consequence ?? "",
    supersededBy: opts.supersededBy ?? "",
    tags: [],
    affectedPaths: [],
    notes: opts.note ?? "",
  };
  for (const tag of opts.tag ?? []) appendSplit(spec.tags, tag);
  for (const ap of opts.affectedPath ?? []) appendSplit(spec.affectedPaths, ap);

  let normalizedStatus: string;
  try {
    normalizedStatus = normalizeDecisionStatus(spec.status);
  } catch (err) {
    console.error(`error: --status ${(err as Error).message}`);
    return EXIT_USAGE;
  }
  spec.status = normalizedStatus;

  if (spec.id.trim() === "") {
    console.error("error: --id is required");
    return EXIT_USAGE;
  }
  if (spec.decision.trim() === "") {
    console.error("error: --decision is required");
    return EXIT_USAGE;
  }
  if (spec.rationale.trim() === "") {
    console.error("error: --rationale is required");
    return EXIT_USAGE;
  }

  const record = buildDecisionRecord(spec);

  let outPath = opts.output ?? "";
  if (outPath === "") {
    outPath = defaultDecisionOutputPath(record);
    try {
      await ensureDecisionOutputDir(outPath, true);
    } catch (err) {
      console.error(`error: ${(err as Error).message}`);
      return EXIT_ERROR;
    }
  } else {
    try {
      await ensureDecisionOutputDir(outPath, false);
    } catch (err) {
      console.error(`error: ${(err as Error).message}`);
      return EXIT_ERROR;
    }
  }

  const jsonMode = Boolean(opts.json) || isJsonExtension(outPath);
  try {
    await writeDecisionRecord(outPath, record, jsonMode);
  } catch (err) {
    console.error(`error: ${(err as Error).message}`);
    return EXIT_ERROR;
  }

  if (jsonMode) {
    renderRecordJson({ path: outPath, record });
  } else {
    renderRecordText(outPath);
  }
  return EXIT_OK;
}

function renderListText(dir: string, entries: DecisionListEntry[]): void {
  console.log(`Directory: ${dir}`);
  console.log(`Count: ${entries.length}`);
  if (entries.length === 0) return;
  console.log("Records:");
  for (const rec of entries) {
    console.log(
      `  - id=${rec.id} status=${rec.status} title=${JSON.stringify(rec.title)}`
    );
    console.log(`    decision: ${rec.decision}`);
    console.log(`    path: ${rec.path}`);
  }
}

async function runListAction(opts: ListActionOptions): Promise<number> {
  const dir = opts.dir ?? DECISION_RECORD_DEFAULT_DIR;
  let entries: DecisionListEntry[];
  try {
    entries = await listDecisionRecords(dir);
  } catch (err) {
    console.error(`error: ${(err as Error).message}`);
    return EXIT_ERROR;
  }

  if (opts.json) {
    console.log(
      JSON.stringify(
        { directory: dir, count: entries.length, records: entries },
        null,
        2
      )
    );
    return EXIT_OK;
  }
  renderListText(dir, entries);
  return EXIT_OK;
}

function renderQueryText(
  dir: string,
  keyword: string,
  matches: DecisionListEntry[]
): void {
  console.log(`Directory: ${dir}`);
  console.log(`Keyword: ${JSON.stringify(keyword)}`);
  console.log(`Count: ${matches.length}`);
  if (matches.length === 0) return;
  console.log("Records:");
  for (const rec of matches) {
    console.log(
      `  - id=${rec.id} status=${rec.status} title=${JSON.stringify(rec.title)}`
    );
    console.log(`    decision: ${rec.decision}`);
    console.log(`    path: ${rec.path}`);
  }
}

async function runQueryAction(opts: QueryActionOptions): Promise<number> {
  const dir = opts.dir ?? DECISION_RECORD_DEFAULT_DIR;
  const keyword = (opts.keyword ?? "").trim();
  if (keyword === "") {
    console.error("error: --keyword is required");
    return EXIT_USAGE;
  }

  let entries: DecisionListEntry[];
  try {
    entries = await listDecisionRecords(dir);
  } catch (err) {
    console.error(`error: ${(err as Error).message}`);
    return EXIT_ERROR;
  }

  const matches = matchDecisionQuery(entries, keyword);

  if (opts.json) {
    console.log(
      JSON.stringify(
        { directory: dir, keyword, count: matches.length, records: matches },
        null,
        2
      )
    );
    return EXIT_OK;
  }
  renderQueryText(dir, keyword, matches);
  return EXIT_OK;
}

function renderAffectedText(
  dir: string,
  target: string,
  matches: DecisionListEntry[]
): void {
  console.log(`Directory: ${dir}`);
  console.log(`Path: ${target}`);
  console.log(`Count: ${matches.length}`);
  if (matches.length === 0) return;
  console.log("Records:");
  for (const rec of matches) {
    console.log(
      `  - id=${rec.id} status=${rec.status} title=${JSON.stringify(rec.title)}`
    );
    console.log(`    decision: ${rec.decision}`);
    console.log(`    path: ${rec.path}`);
  }
}

async function runAffectedAction(opts: AffectedActionOptions): Promise<number> {
  const dir = opts.dir ?? DECISION_RECORD_DEFAULT_DIR;
  const target = (opts.path ?? "").trim();
  if (target === "") {
    console.error("error: --path is required");
    return EXIT_USAGE;
  }

  let entries: DecisionListEntry[];
  try {
    entries = await listDecisionRecords(dir);
  } catch (err) {
    console.error(`error: ${(err as Error).message}`);
    return EXIT_ERROR;
  }

  const matches = matchDecisionAffected(entries, target);

  if (opts.json) {
    console.log(
      JSON.stringify(
        {
          directory: dir,
          path: target,
          count: matches.length,
          records: matches,
        },
        null,
        2
      )
    );
    return EXIT_OK;
  }
  renderAffectedText(dir, target, matches);
  return EXIT_OK;
}

async function runLinkAction(opts: LinkActionOptions): Promise<number> {
  const cardPath = (opts.card ?? "").trim();
  if (cardPath === "") {
    console.error("error: --card is required");
    return EXIT_USAGE;
  }
  const decisions: string[] = [];
  for (const d of opts.decision ?? []) appendSplit(decisions, d);
  if (decisions.length === 0) {
    console.error("error: --decision is required");
    return EXIT_USAGE;
  }

  let doc: Record<string, unknown>;
  try {
    doc = loadCompletionCardForLink(cardPath);
  } catch (err) {
    console.error(`error: ${(err as Error).message}`);
    return EXIT_ERROR;
  }

  let result: ReturnType<typeof applyDecisionLinkRefs>;
  try {
    result = applyDecisionLinkRefs(doc, decisions);
  } catch (err) {
    console.error(`error: ${(err as Error).message}`);
    return EXIT_ERROR;
  }

  const dest = (opts.out ?? "").trim() || cardPath;
  if (dest !== cardPath) {
    try {
      await ensureDecisionOutputDir(dest, false);
    } catch (err) {
      console.error(`error: ${(err as Error).message}`);
      return EXIT_ERROR;
    }
  }

  const useJson = Boolean(opts.json) || isJsonExtension(dest);
  const fs = await import("node:fs/promises");
  try {
    if (useJson) {
      await fs.writeFile(dest, `${JSON.stringify(result.doc, null, 2)}\n`);
    } else {
      const { default: yaml } = await import("yaml");
      await fs.writeFile(dest, yaml.stringify(result.doc));
    }
  } catch (err) {
    console.error(`error: ${(err as Error).message}`);
    return EXIT_ERROR;
  }

  if (opts.json) {
    const out = {
      schema_version: DECISION_LINK_RESULT_SCHEMA_VERSION,
      card: cardPath,
      out: dest,
      added: result.added,
      skipped: result.skipped,
      decision_refs: collectDecisionLinkRefs(result.doc),
    };
    console.log(JSON.stringify(out, null, 2));
    return EXIT_OK;
  }

  console.log(`Card: ${cardPath}`);
  console.log(`Output: ${dest}`);
  console.log(`Added: ${formatDecisionLinkList(result.added)}`);
  console.log(`Skipped: ${formatDecisionLinkList(result.skipped)}`);
  console.log(
    `Total decision refs: ${collectDecisionLinkRefs(result.doc).length}`
  );
  return EXIT_OK;
}

const RECORD_USAGE =
  "usage: xh decision record --id <id> --decision <text> --rationale <text> [--title <text>] [--status proposed|accepted|superseded|deprecated] [--date <iso-date>] [--context <text>] [--consequence <text>] [--superseded-by <id>] [--tag <text> ...] [--affected-path <path> ...] [--note <text>] [--output <path>] [--json]";

const QUERY_USAGE =
  "usage: xh decision query --keyword <text> [--dir <path>] [--json]";

const AFFECTED_USAGE =
  "usage: xh decision affected --path <path> [--dir <path>] [--json]";

const LINK_USAGE =
  "usage: xh decision link --card <path> --decision <id> [--decision <id> ...] [--out <path>] [--json]";

function recordCommand(): Command {
  const collectStrings = (value: string, previous: string[] = []): string[] => [
    ...previous,
    value,
  ];
  return new Command("record")
    .description("Write a decision record (ADR-lite) to disk")
    .option("--id <id>", "Stable id for the decision record")
    .option("--title <text>", "Short human-readable title")
    .option("--date <iso-date>", "ISO-8601 date the decision was recorded")
    .option(
      "--status <status>",
      `Lifecycle status (one of ${DECISION_RECORD_STATUSES.join(", ")})`,
      "proposed"
    )
    .option("--decision <text>", "Plain-language statement of the decision")
    .option("--rationale <text>", "Reasoning behind the decision")
    .option("--context <text>", "Optional situational context")
    .option("--consequence <text>", "Optional description of consequences")
    .option(
      "--superseded-by <id>",
      "Identifier of a later superseding decision"
    )
    .option(
      "--tag <text>",
      "Free-form tag (repeatable, also accepts comma-delimited values)",
      collectStrings,
      [] as string[]
    )
    .option(
      "--affected-path <path>",
      "Source path affected by the decision (repeatable, also accepts comma-delimited values)",
      collectStrings,
      [] as string[]
    )
    .option("--note <text>", "Free-form notes")
    .option(
      "--output <path>",
      "Override the output path (default: decisions/<id>.yaml)"
    )
    .option("--json", "Emit JSON output (default: YAML)", false)
    .action(async (opts: RecordActionOptions) => {
      const code = await runRecordAction(opts);
      if (code !== EXIT_OK) {
        if (code === EXIT_USAGE) {
          console.error(RECORD_USAGE);
        }
        process.exit(code);
      }
    });
}

function listCommand(): Command {
  return new Command("list")
    .description("List decision records in a directory")
    .option("--dir <path>", "Directory to scan", DECISION_RECORD_DEFAULT_DIR)
    .option("--json", "Emit JSON output", false)
    .action(async (opts: ListActionOptions) => {
      const code = await runListAction(opts);
      if (code !== EXIT_OK) process.exit(code);
    });
}

function queryCommand(): Command {
  return new Command("query")
    .description("Search decision records by keyword")
    .option("--dir <path>", "Directory to scan", DECISION_RECORD_DEFAULT_DIR)
    .option("--keyword <text>", "Substring to search (case-insensitive)")
    .option("--json", "Emit JSON output", false)
    .action(async (opts: QueryActionOptions) => {
      const code = await runQueryAction(opts);
      if (code !== EXIT_OK) {
        if (code === EXIT_USAGE) {
          console.error(QUERY_USAGE);
        }
        process.exit(code);
      }
    });
}

function affectedCommand(): Command {
  return new Command("affected")
    .description("Find decision records affecting a given path")
    .option("--dir <path>", "Directory to scan", DECISION_RECORD_DEFAULT_DIR)
    .option("--path <path>", "Path or glob to match against affected_paths")
    .option("--json", "Emit JSON output", false)
    .action(async (opts: AffectedActionOptions) => {
      const code = await runAffectedAction(opts);
      if (code !== EXIT_OK) {
        if (code === EXIT_USAGE) {
          console.error(AFFECTED_USAGE);
        }
        process.exit(code);
      }
    });
}

function linkCommand(): Command {
  const collectStrings = (value: string, previous: string[] = []): string[] => [
    ...previous,
    value,
  ];
  return new Command("link")
    .description(
      "Append decision ids to a completion card's context_alignment.decision_refs"
    )
    .option("--card <path>", "Path to completion card YAML/JSON")
    .option(
      "--decision <id>",
      "Decision id to link (repeatable; comma-delimited also accepted)",
      collectStrings,
      [] as string[]
    )
    .option("--out <path>", "Output path (default: in-place)")
    .option("--json", "Emit JSON output", false)
    .action(async (opts: LinkActionOptions) => {
      const code = await runLinkAction(opts);
      if (code !== EXIT_OK) {
        if (code === EXIT_USAGE) {
          console.error(LINK_USAGE);
        }
        process.exit(code);
      }
    });
}

export function decisionCommand(): Command {
  const cmd = new Command("decision").description(
    "File-first decision memory (ADR-lite) helpers"
  );
  cmd.addCommand(recordCommand());
  cmd.addCommand(listCommand());
  cmd.addCommand(queryCommand());
  cmd.addCommand(affectedCommand());
  cmd.addCommand(linkCommand());
  return cmd;
}

// Internal exports used by focused unit tests.
export const __test = {
  runRecordAction: (opts: RecordActionOptions) => runRecordAction(opts),
  runListAction: (opts: ListActionOptions) => runListAction(opts),
  runQueryAction: (opts: QueryActionOptions) => runQueryAction(opts),
  runAffectedAction: (opts: AffectedActionOptions) => runAffectedAction(opts),
  runLinkAction: (opts: LinkActionOptions) => runLinkAction(opts),
  buildDecisionRecord,
  normalizeDecisionStatus,
  defaultDecisionOutputPath,
  matchDecisionAffected,
  matchDecisionQuery,
  applyDecisionLinkRefs,
  collectDecisionLinkRefs,
  isJsonExtension,
  DECISION_RECORD_DEFAULT_DIR,
  DECISION_LINK_RESULT_SCHEMA_VERSION,
};

// Suppress unused import warnings from the helpers retained for parity.
void readFlag;
void path;
void serializeDecisionRecord;
