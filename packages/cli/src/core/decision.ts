import * as path from "node:path";
import fs from "fs-extra";
import yaml from "yaml";

export const DECISION_RECORD_SCHEMA_VERSION = "1";
export const DECISION_RECORD_DEFAULT_DIR = "decisions";
export const DECISION_LINK_RESULT_SCHEMA_VERSION = "x-harness.decision.link.v1";

export const DECISION_RECORD_STATUSES = [
  "proposed",
  "accepted",
  "superseded",
  "deprecated",
] as const;

export type DecisionRecordStatus = (typeof DECISION_RECORD_STATUSES)[number];

export interface DecisionRecordSpec {
  id: string;
  title: string;
  date: string;
  status: string;
  decision: string;
  rationale: string;
  context: string;
  consequences: string;
  supersededBy: string;
  tags: string[];
  affectedPaths: string[];
  notes: string;
}

export interface DecisionRecord {
  schema_version: string;
  id: string;
  decision: string;
  rationale: string;
  title?: string;
  date?: string;
  status?: string;
  context?: string;
  consequences?: string;
  superseded_by?: string;
  tags?: string[];
  affected_paths?: string[];
  notes?: string;
}

export interface DecisionListEntry {
  id: string;
  status: string;
  title: string;
  decision: string;
  path: string;
}

function appendListUnique(list: string[], value: string): string[] {
  if (value === "" || list.includes(value)) {
    return list;
  }
  return [...list, value];
}

function splitListValue(value: string): string[] {
  return value
    .split(",")
    .map((part) => part.trim())
    .filter((part) => part !== "");
}

function collectListField(
  spec: DecisionRecordSpec,
  key: "tags" | "affectedPaths",
  raw: string
): void {
  const values = splitListValue(raw);
  for (const v of values) {
    spec[key] = appendListUnique(spec[key], v);
  }
}

export function buildDecisionRecord(spec: DecisionRecordSpec): DecisionRecord {
  const record: DecisionRecord = {
    schema_version: DECISION_RECORD_SCHEMA_VERSION,
    id: spec.id.trim(),
    decision: spec.decision.trim(),
    rationale: spec.rationale.trim(),
  };
  const title = spec.title.trim();
  if (title) record.title = title;
  const date = spec.date.trim();
  if (date) {
    record.date = date;
  } else {
    record.date = todayIsoDate();
  }
  const status = spec.status.trim();
  if (status) record.status = status;
  const context = spec.context.trim();
  if (context) record.context = context;
  const consequences = spec.consequences.trim();
  if (consequences) record.consequences = consequences;
  const supersededBy = spec.supersededBy.trim();
  if (supersededBy) record.superseded_by = supersededBy;
  if (spec.tags.length > 0) record.tags = [...spec.tags];
  if (spec.affectedPaths.length > 0)
    record.affected_paths = [...spec.affectedPaths];
  const notes = spec.notes.trim();
  if (notes) record.notes = notes;
  return record;
}

export function normalizeDecisionStatus(raw: string): string {
  const trimmed = raw.trim().toLowerCase();
  if (trimmed === "") return "proposed";
  for (const candidate of DECISION_RECORD_STATUSES) {
    if (candidate === trimmed) return candidate;
  }
  throw new Error(
    `expected one of ${DECISION_RECORD_STATUSES.join(", ")}, got "${raw}"`
  );
}

export function defaultDecisionOutputPath(record: DecisionRecord): string {
  return path.join(DECISION_RECORD_DEFAULT_DIR, `${record.id}.yaml`);
}

export async function ensureDecisionOutputDir(
  outPath: string,
  allowCreate: boolean
): Promise<void> {
  const abs = path.resolve(outPath);
  const parent = path.dirname(abs);
  const parentExists = await fs.pathExists(parent);
  if (parentExists) return;
  if (!allowCreate) {
    throw new Error(`parent directory does not exist: ${parent}`);
  }
  await fs.mkdir(parent, { recursive: true });
}

export function isValidDecisionEnforce(value: string): boolean {
  return value === "off" || value === "advisory" || value === "block";
}

export interface FlagSpecParseInput {
  args: string[];
  required: (key: string) => string | null;
  unknownFlag: (flag: string) => never;
  unknownArg: (arg: string) => never;
  onAssign?: (key: string, value: string) => void;
  onRepeatedAssign?: (key: string, value: string) => void;
  onBool?: (key: string) => void;
}

export function nextFlagValue(
  args: string[],
  i: number,
  flag: string
): { value: string; next: number } {
  if (i + 1 >= args.length) {
    throw new Error(`error: ${flag} requires a value`);
  }
  return { value: args[i + 1], next: i + 2 };
}

function readDecisionFile(filePath: string): Record<string, unknown> {
  const text = fs.readFileSync(filePath, "utf-8");
  const parsed = yaml.parse(text);
  if (parsed == null || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(`${filePath}: expected a mapping`);
  }
  return parsed as Record<string, unknown>;
}

function asString(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return JSON.stringify(value);
}

export async function listDecisionRecords(
  dir: string
): Promise<DecisionListEntry[]> {
  const abs = path.resolve(dir);
  const exists = await fs.pathExists(abs);
  if (!exists) return [];
  const stat = await fs.stat(abs);
  if (!stat.isDirectory()) {
    throw new Error(`not a directory: ${abs}`);
  }
  const entries = await fs.readdir(abs, { withFileTypes: true });
  const out: DecisionListEntry[] = [];
  for (const entry of entries) {
    if (entry.isDirectory()) continue;
    const ext = path.extname(entry.name).toLowerCase();
    if (ext !== ".yaml" && ext !== ".yml" && ext !== ".json") continue;
    const full = path.join(abs, entry.name);
    const doc = readDecisionFile(full);
    out.push({
      id: asString(doc.id),
      status: asString(doc.status),
      title: asString(doc.title),
      decision: asString(doc.decision),
      path: full,
    });
  }
  out.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
  return out;
}

export function decisionRecordSearchableText(
  doc: Record<string, unknown>
): string {
  const fields: string[] = [];
  for (const key of [
    "id",
    "title",
    "decision",
    "rationale",
    "context",
    "consequences",
    "notes",
  ]) {
    const value = doc[key];
    if (typeof value === "string" && value !== "") {
      fields.push(value);
    }
  }
  const tags = doc.tags;
  if (Array.isArray(tags)) {
    for (const t of tags) {
      if (typeof t === "string" && t !== "") {
        fields.push(t);
      }
    }
  }
  const affected = doc.affected_paths;
  if (Array.isArray(affected)) {
    for (const p of affected) {
      if (typeof p === "string" && p !== "") {
        fields.push(p);
      }
    }
  }
  return fields.join(" ").toLowerCase();
}

export function matchDecisionQuery(
  entries: DecisionListEntry[],
  keyword: string
): DecisionListEntry[] {
  const needle = keyword.trim().toLowerCase();
  if (needle === "") return [];
  const out: DecisionListEntry[] = [];
  for (const entry of entries) {
    const doc = readDecisionFile(entry.path);
    if (decisionRecordSearchableText(doc).includes(needle)) {
      out.push(entry);
    }
  }
  return out;
}

export function matchDecisionAffected(
  entries: DecisionListEntry[],
  target: string
): DecisionListEntry[] {
  const cleaned = path.posix.normalize(target.trim());
  const cleanedWin = path.normalize(target.trim());
  const candidate = cleaned === "." ? "" : cleaned;
  const candidateWin = cleanedWin === "." ? "" : cleanedWin;
  if (candidate === "" && candidateWin === "") return [];
  const out: DecisionListEntry[] = [];
  for (const entry of entries) {
    const doc = readDecisionFile(entry.path);
    if (decisionRecordMatchesPath(doc, candidate, candidateWin)) {
      out.push(entry);
    }
  }
  return out;
}

function decisionRecordMatchesPath(
  doc: Record<string, unknown>,
  posixTarget: string,
  winTarget: string
): boolean {
  const raw = doc.affected_paths;
  if (!Array.isArray(raw)) return false;
  for (const p of raw) {
    if (typeof p !== "string") continue;
    const pattern = p.trim();
    if (pattern === "") continue;
    const posixPattern = path.posix.normalize(pattern);
    const winPattern = path.normalize(pattern);
    if (
      globMatch(posixPattern, posixTarget) ||
      globMatch(winPattern, winTarget)
    ) {
      return true;
    }
  }
  return false;
}

function globMatch(pattern: string, target: string): boolean {
  if (pattern === "" || target === "") return false;
  if (pattern === target) return true;
  // Convert simple glob to a regex: ** matches across separators, * matches
  // within a single segment, ? matches a single char, others literal.
  const regex = globToRegExp(pattern);
  return regex.test(target);
}

function globToRegExp(pattern: string): RegExp {
  let body = "";
  for (let i = 0; i < pattern.length; i += 1) {
    const ch = pattern[i];
    if (ch === "*") {
      if (pattern[i + 1] === "*") {
        body += ".*";
        i += 1;
        // swallow a following separator so **/x matches x too
        if (pattern[i + 1] === "/" || pattern[i + 1] === path.sep) {
          i += 1;
          body += "(?:.*/)?";
        }
      } else {
        body += "[^/]*";
      }
    } else if (ch === "?") {
      body += "[^/]";
    } else if (/[.+^$|()[\]{}\\]/.test(ch)) {
      body += `\\${ch}`;
    } else {
      body += ch;
    }
  }
  return new RegExp(`^${body}$`);
}

export function loadCompletionCardForLink(
  filePath: string
): Record<string, unknown> {
  const abs = path.resolve(filePath);
  if (!fs.existsSync(abs)) {
    throw new Error(`card not found: ${filePath}`);
  }
  const text = fs.readFileSync(abs, "utf-8");
  if (text.trim() === "") {
    throw new Error(`card is empty: ${filePath}`);
  }
  const parsed = yaml.parse(text);
  if (parsed == null || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(`card is not a YAML/JSON mapping: ${filePath}`);
  }
  return parsed as Record<string, unknown>;
}

export interface DecisionLinkResult {
  doc: Record<string, unknown>;
  added: string[];
  skipped: string[];
}

export function applyDecisionLinkRefs(
  doc: Record<string, unknown>,
  inputs: string[]
): DecisionLinkResult {
  const caRaw = doc.context_alignment;
  let ca: Record<string, unknown>;
  if (caRaw != null) {
    if (typeof caRaw !== "object" || Array.isArray(caRaw)) {
      throw new Error(
        `context_alignment must be a mapping, got ${typeof caRaw}`
      );
    }
    ca = caRaw as Record<string, unknown>;
  } else {
    ca = {};
    doc.context_alignment = ca;
  }

  const refsRaw = ca.decision_refs;
  let refs: unknown[];
  if (refsRaw == null) {
    refs = [];
    ca.decision_refs = refs;
  } else if (Array.isArray(refsRaw)) {
    refs = refsRaw;
  } else {
    throw new Error(
      `context_alignment.decision_refs must be an array, got ${typeof refsRaw}`
    );
  }

  const existing = new Set<string>();
  for (const r of refs) {
    if (typeof r === "string") existing.add(r);
  }

  const added: string[] = [];
  const skipped: string[] = [];
  const seen = new Set<string>();
  for (const candidate of inputs) {
    if (seen.has(candidate)) continue;
    seen.add(candidate);
    if (existing.has(candidate)) {
      skipped.push(candidate);
      continue;
    }
    refs.push(candidate);
    existing.add(candidate);
    added.push(candidate);
  }
  ca.decision_refs = refs;
  return { doc, added, skipped };
}

export function collectDecisionLinkRefs(
  doc: Record<string, unknown>
): string[] {
  const ca = doc.context_alignment;
  if (ca == null || typeof ca !== "object" || Array.isArray(ca)) {
    return [];
  }
  const refs = (ca as Record<string, unknown>).decision_refs;
  if (!Array.isArray(refs)) return [];
  const out: string[] = [];
  for (const v of refs) {
    if (typeof v === "string") out.push(v);
  }
  return out;
}

export function formatDecisionLinkList(items: string[]): string {
  if (items.length === 0) return "(none)";
  return items.join(", ");
}

export async function writeDecisionRecord(
  outPath: string,
  record: DecisionRecord,
  jsonMode: boolean
): Promise<void> {
  let body: string;
  const ext = path.extname(outPath).toLowerCase();
  if (jsonMode || ext === ".json") {
    body = `${JSON.stringify(record, null, 2)}\n`;
  } else {
    body = `${yaml.stringify(record)}`;
  }
  await fs.writeFile(outPath, body, "utf-8");
}

export function serializeDecisionRecord(
  record: DecisionRecord,
  jsonMode: boolean
): string {
  if (jsonMode) {
    return `${JSON.stringify(record, null, 2)}\n`;
  }
  return yaml.stringify(record);
}

function todayIsoDate(): string {
  const now = new Date();
  const y = now.getUTCFullYear();
  const m = String(now.getUTCMonth() + 1).padStart(2, "0");
  const d = String(now.getUTCDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

export function isJsonExtension(outPath: string): boolean {
  return path.extname(outPath).toLowerCase() === ".json";
}

export { collectListField as collectList, readDecisionFile };
