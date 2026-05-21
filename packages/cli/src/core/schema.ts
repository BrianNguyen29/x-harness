import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import fs from "fs-extra";
import * as YAML from "yaml";
import { Ajv2020 } from "ajv/dist/2020.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const ajv = new Ajv2020({ strict: false });

export async function loadSchema(name: string): Promise<Record<string, unknown>> {
  const schemaPath = join(__dirname, "..", "..", "..", "..", "schemas", `${name}.schema.json`);
  if (!(await fs.pathExists(schemaPath))) {
    throw new Error(`Schema not found: ${schemaPath}`);
  }
  return fs.readJson(schemaPath);
}

export function compileSchema(schema: Record<string, unknown>) {
  return ajv.compile(schema);
}

export async function readYamlOrJson(filePath: string): Promise<unknown> {
  const content = await fs.readFile(filePath, "utf-8");
  if (filePath.endsWith(".yaml") || filePath.endsWith(".yml")) {
    return YAML.parse(content);
  }
  if (filePath.endsWith(".json")) {
    return JSON.parse(content);
  }
  // Try YAML first, then JSON
  try {
    return YAML.parse(content);
  } catch {
    return JSON.parse(content);
  }
}

export async function readJsonl(filePath: string): Promise<Record<string, unknown>[]> {
  if (!(await fs.pathExists(filePath))) return [];
  const content = await fs.readFile(filePath, "utf-8");
  return content
    .split("\n")
    .map((line: string) => line.trim())
    .filter((line: string) => line.length > 0)
    .map((line: string) => JSON.parse(line));
}
