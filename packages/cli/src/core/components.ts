import { execFile } from "node:child_process";
import * as path from "node:path";
import { compileSchema, loadSchema, readYamlOrJson } from "./schema.js";
import { loadAuthorityPolicy } from "./authority.js";

export interface ComponentEntry {
  id: string;
  kind: string;
  paths: string[];
  owner: string;
  stability: "experimental" | "stable" | "deprecated";
  agent_edit: "agent_editable" | "human_approved" | "human_only";
  tests: string[];
  description?: string;
}

export interface ComponentsRegistry {
  version: number;
  components: ComponentEntry[];
}

export interface ComponentValidationResult {
  ok: boolean;
  errors: string[];
  warnings: string[];
  component_count: number;
  protected_paths_checked: number;
  protected_paths_covered: number;
}

export interface ComponentChange {
  component: ComponentEntry;
  files: string[];
}

export interface ComponentChangeResult {
  files: string[];
  components: ComponentChange[];
  unregistered_files: string[];
}

export function registryPath(root: string): string {
  return path.join(root, "components", "registry.yaml");
}

export async function loadComponentsRegistry(
  root: string
): Promise<ComponentsRegistry> {
  const filePath = registryPath(root);
  const data = (await readYamlOrJson(filePath)) as ComponentsRegistry;
  return data;
}

function normalizePath(filePath: string): string {
  return filePath.replace(/\\/g, "/").replace(/^\.\//, "");
}

function globToRegex(pattern: string): RegExp {
  const normalized = normalizePath(pattern)
    .replace(/[.+^${}()|[\]\\]/g, "\\$&")
    .replace(/\*\*/g, "{{DOUBLE_STAR}}")
    .replace(/\*/g, "[^/]*")
    .replace(/\{\{DOUBLE_STAR\}\}/g, ".*");
  return new RegExp(`^${normalized}$`);
}

export function componentPathMatches(
  pattern: string,
  filePath: string
): boolean {
  return globToRegex(pattern).test(normalizePath(filePath));
}

export function componentPathCoversPattern(
  componentPattern: string,
  protectedPattern: string
): boolean {
  const component = normalizePath(componentPattern);
  const protectedPath = normalizePath(protectedPattern);
  if (component === protectedPath) return true;
  if (component.endsWith("/**")) {
    const prefix = component.slice(0, -3);
    return protectedPath === prefix || protectedPath.startsWith(`${prefix}/`);
  }
  if (!protectedPath.includes("*")) {
    return componentPathMatches(component, protectedPath);
  }
  return false;
}

export async function validateComponentsRegistry(
  root: string
): Promise<ComponentValidationResult> {
  const errors: string[] = [];
  const warnings: string[] = [];
  let registry: ComponentsRegistry | null = null;

  try {
    registry = await loadComponentsRegistry(root);
  } catch (err) {
    return {
      ok: false,
      errors: [
        `components registry load error: ${
          err instanceof Error ? err.message : String(err)
        }`,
      ],
      warnings,
      component_count: 0,
      protected_paths_checked: 0,
      protected_paths_covered: 0,
    };
  }

  try {
    const schema = await loadSchema("components-registry");
    const validate = compileSchema(schema);
    if (!validate(registry)) {
      const schemaErrors = validate.errors?.map(
        (e) => `${e.instancePath || "/"} ${e.message}`
      ) ?? ["validation failed"];
      errors.push(
        `components registry schema validation failed: ${schemaErrors.join("; ")}`
      );
    }
  } catch (err) {
    errors.push(
      `components registry schema error: ${
        err instanceof Error ? err.message : String(err)
      }`
    );
  }

  const seenIds = new Set<string>();
  for (const component of registry.components ?? []) {
    if (seenIds.has(component.id)) {
      errors.push(`duplicate component id: ${component.id}`);
    }
    seenIds.add(component.id);
  }

  let protectedPathsChecked = 0;
  let protectedPathsCovered = 0;
  try {
    const authority = await loadAuthorityPolicy(root);
    for (const protectedPath of authority.protected_paths) {
      protectedPathsChecked += 1;
      const covered = registry.components.some((component) =>
        component.paths.some((componentPath) =>
          componentPathCoversPattern(componentPath, protectedPath.path)
        )
      );
      if (covered) {
        protectedPathsCovered += 1;
      } else {
        errors.push(
          `protected path is not registered to any component: ${protectedPath.path}`
        );
      }
    }
  } catch (err) {
    errors.push(
      `authority policy coverage check failed: ${
        err instanceof Error ? err.message : String(err)
      }`
    );
  }

  return {
    ok: errors.length === 0,
    errors,
    warnings,
    component_count: registry.components?.length ?? 0,
    protected_paths_checked: protectedPathsChecked,
    protected_paths_covered: protectedPathsCovered,
  };
}

export function findComponentsForFile(
  registry: ComponentsRegistry,
  filePath: string
): ComponentEntry[] {
  const normalized = normalizePath(filePath);
  return registry.components.filter((component) =>
    component.paths.some((componentPath) =>
      componentPathMatches(componentPath, normalized)
    )
  );
}

export function explainComponent(
  registry: ComponentsRegistry,
  id: string
): ComponentEntry | null {
  return registry.components.find((component) => component.id === id) ?? null;
}

function execGit(
  args: string[],
  cwd: string
): Promise<{ stdout: string; stderr: string }> {
  return new Promise((resolve, reject) => {
    execFile("git", args, { cwd }, (error, stdout, stderr) => {
      if (error) {
        reject(new Error(stderr.trim() || error.message));
        return;
      }
      resolve({ stdout, stderr });
    });
  });
}

export async function listChangedFilesFromGit(
  root: string,
  base: string
): Promise<string[]> {
  try {
    const { stdout } = await execGit(
      ["diff", "--name-only", `${base}...HEAD`],
      root
    );
    return stdout
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean);
  } catch {
    const { stdout } = await execGit(
      ["diff", "--name-only", base, "HEAD"],
      root
    );
    return stdout
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean);
  }
}

export function classifyChangedFiles(
  registry: ComponentsRegistry,
  files: string[]
): ComponentChangeResult {
  const componentFiles = new Map<
    string,
    { component: ComponentEntry; files: Set<string> }
  >();
  const unregisteredFiles: string[] = [];

  for (const file of files.map(normalizePath)) {
    const components = findComponentsForFile(registry, file);
    if (components.length === 0) {
      unregisteredFiles.push(file);
      continue;
    }
    for (const component of components) {
      const current = componentFiles.get(component.id) ?? {
        component,
        files: new Set<string>(),
      };
      current.files.add(file);
      componentFiles.set(component.id, current);
    }
  }

  const components = [...componentFiles.values()]
    .map((entry) => ({
      component: entry.component,
      files: [...entry.files].sort(),
    }))
    .sort((a, b) => a.component.id.localeCompare(b.component.id));

  return {
    files: files.map(normalizePath),
    components,
    unregistered_files: unregisteredFiles.sort(),
  };
}
