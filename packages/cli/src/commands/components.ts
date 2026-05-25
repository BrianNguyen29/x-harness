import { Command } from "commander";
import * as path from "node:path";
import {
  classifyChangedFiles,
  explainComponent,
  listChangedFilesFromGit,
  loadComponentsRegistry,
  validateComponentsRegistry,
} from "../core/components.js";

interface ComponentsOptions {
  root?: string;
  json?: boolean;
}

interface ComponentsChangedOptions extends ComponentsOptions {
  base?: string;
  files?: string;
}

function renderValidationText(
  result: Awaited<ReturnType<typeof validateComponentsRegistry>>
): void {
  console.log(`ok: ${result.ok}`);
  console.log(`components: ${result.component_count}`);
  console.log(
    `protected_paths: ${result.protected_paths_covered}/${result.protected_paths_checked} covered`
  );
  if (result.errors.length > 0) {
    console.log("");
    console.log("Errors:");
    for (const error of result.errors) {
      console.log(`  - ${error}`);
    }
  }
  if (result.warnings.length > 0) {
    console.log("");
    console.log("Warnings:");
    for (const warning of result.warnings) {
      console.log(`  - ${warning}`);
    }
  }
}

export async function componentsValidateAction(
  opts: ComponentsOptions
): Promise<void> {
  const root = path.resolve(opts.root ?? process.cwd());
  const result = await validateComponentsRegistry(root);
  if (opts.json) {
    console.log(JSON.stringify(result, null, 2));
  } else {
    renderValidationText(result);
  }
  process.exit(result.ok ? 0 : 1);
}

export async function componentsListAction(
  opts: ComponentsOptions
): Promise<void> {
  const root = path.resolve(opts.root ?? process.cwd());
  const registry = await loadComponentsRegistry(root);
  if (opts.json) {
    console.log(JSON.stringify(registry, null, 2));
    return;
  }
  console.log("# x-harness Components");
  console.log("");
  for (const component of registry.components) {
    console.log(
      `- ${component.id} (${component.kind}, ${component.stability})`
    );
    console.log(`  owner: ${component.owner}`);
    console.log(`  agent_edit: ${component.agent_edit}`);
    console.log(`  paths: ${component.paths.join(", ")}`);
  }
}

export async function componentsExplainAction(
  opts: ComponentsOptions & { id?: string }
): Promise<void> {
  const root = path.resolve(opts.root ?? process.cwd());
  if (!opts.id) {
    console.error("Error: --id <component-id> is required");
    process.exit(2);
  }
  const registry = await loadComponentsRegistry(root);
  const component = explainComponent(registry, opts.id);
  if (!component) {
    console.error(`Error: component not found: ${opts.id}`);
    process.exit(1);
  }
  if (opts.json) {
    console.log(JSON.stringify(component, null, 2));
    return;
  }
  console.log(`Component: ${component.id}`);
  console.log(`Kind: ${component.kind}`);
  console.log(`Owner: ${component.owner}`);
  console.log(`Stability: ${component.stability}`);
  console.log(`Agent edit: ${component.agent_edit}`);
  console.log("");
  console.log("Paths:");
  for (const componentPath of component.paths) {
    console.log(`  - ${componentPath}`);
  }
  console.log("");
  console.log("Tests:");
  for (const test of component.tests) {
    console.log(`  - ${test}`);
  }
}

export async function componentsChangedAction(
  opts: ComponentsChangedOptions
): Promise<void> {
  const root = path.resolve(opts.root ?? process.cwd());
  const registry = await loadComponentsRegistry(root);
  let files: string[];
  const base = opts.base ?? "main";

  if (opts.files && opts.files.trim().length > 0) {
    files = opts.files
      .split(",")
      .map((file) => file.trim())
      .filter(Boolean);
  } else {
    try {
      files = await listChangedFilesFromGit(root, base);
    } catch (err) {
      console.error(
        `Error reading changed files: ${
          err instanceof Error ? err.message : String(err)
        }`
      );
      process.exit(2);
    }
  }

  const result = classifyChangedFiles(registry, files);
  if (opts.json) {
    console.log(
      JSON.stringify(
        {
          base,
          ...result,
          components: result.components.map((entry) => ({
            id: entry.component.id,
            kind: entry.component.kind,
            owner: entry.component.owner,
            stability: entry.component.stability,
            agent_edit: entry.component.agent_edit,
            files: entry.files,
            tests: entry.component.tests,
          })),
        },
        null,
        2
      )
    );
    return;
  }

  console.log(`Changed files: ${result.files.length}`);
  console.log(`Components touched: ${result.components.length}`);
  for (const entry of result.components) {
    console.log("");
    console.log(`- ${entry.component.id}`);
    for (const file of entry.files) {
      console.log(`  - ${file}`);
    }
  }
  if (result.unregistered_files.length > 0) {
    console.log("");
    console.log("Unregistered files:");
    for (const file of result.unregistered_files) {
      console.log(`  - ${file}`);
    }
  }
}

export function componentsCommand(): Command {
  return new Command("components")
    .description("Inspect and validate the x-harness component registry")
    .addCommand(
      new Command("validate")
        .description(
          "Validate components/registry.yaml and protected-path coverage"
        )
        .option("--json", "Output JSON", false)
        .option("--root <path>", "Repository root", process.cwd())
        .action(componentsValidateAction)
    )
    .addCommand(
      new Command("list")
        .description("List registered harness components")
        .option("--json", "Output JSON", false)
        .option("--root <path>", "Repository root", process.cwd())
        .action(componentsListAction)
    )
    .addCommand(
      new Command("explain")
        .description("Explain a registered component")
        .requiredOption("--id <component-id>", "Component id")
        .option("--json", "Output JSON", false)
        .option("--root <path>", "Repository root", process.cwd())
        .action(componentsExplainAction)
    )
    .addCommand(
      new Command("changed")
        .description("Map changed files to registered components")
        .option("--base <ref>", "Git base ref for diff", "main")
        .option(
          "--files <paths>",
          "Comma-separated changed files (bypasses git diff)"
        )
        .option("--json", "Output JSON", false)
        .option("--root <path>", "Repository root", process.cwd())
        .action(componentsChangedAction)
    );
}
