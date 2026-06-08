import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import {
  getCanonicalContext,
  getContextHash,
  generateManagedBlock,
  injectManagedBlock,
  validateManagedBlock,
} from "../core/context.js";
import {
  generateManagedContractBlock,
  injectManagedContractBlock,
  loadRuntimeContract,
  MANAGED_CONTRACT_TARGETS,
  renderRuntimeContractMarkdown,
} from "../core/contract.js";
import { checkStaleness, getSourceOfTruthFiles } from "../core/staleness.js";
import {
  generateManifest,
  checkManifest,
  readManifest,
  validateManifest,
  writeManifest,
} from "../core/context-manifest.js";

interface ContextOptions {
  verbose?: boolean;
  json?: boolean;
  refresh?: boolean;
  check?: boolean;
  contract?: boolean;
  writeContractAssets?: boolean;
  root?: string;
}

interface StalenessOptions {
  json?: boolean;
  root?: string;
}

export async function contextCheckAction(opts: ContextOptions): Promise<void> {
  const root = path.resolve(opts.root ?? process.cwd());
  const agentsPath = path.join(root, "AGENTS.md");

  if (!(await fs.pathExists(agentsPath))) {
    console.error(`Error: AGENTS.md not found at ${agentsPath}`);
    process.exit(2);
  }

  const agentsContent = await fs.readFile(agentsPath, "utf-8");
  const validation = validateManagedBlock(agentsContent);

  if (opts.json) {
    const linkedFiles = getSourceOfTruthFiles();
    const missingLinked = [];
    for (const f of linkedFiles) {
      if (!(await fs.pathExists(path.join(root, f)))) {
        missingLinked.push(f);
      }
    }

    console.log(
      JSON.stringify(
        {
          valid: validation.valid,
          note: validation.note,
          missing_linked_files: missingLinked,
        },
        null,
        2
      )
    );
    process.exit(validation.valid ? 0 : 1);
  }

  if (validation.valid) {
    console.log("✓ AGENTS.md managed context block is valid");
  } else {
    console.error(`✗ ${validation.note}`);
    process.exit(1);
  }
}

export async function contextStalenessAction(
  opts: StalenessOptions
): Promise<void> {
  const root = path.resolve(opts.root ?? process.cwd());
  const result = await checkStaleness(root);

  if (opts.json) {
    console.log(
      JSON.stringify(
        {
          stale: result.stale,
          findings: result.findings,
          source_of_truth_files: getSourceOfTruthFiles(),
        },
        null,
        2
      )
    );
    process.exit(result.stale ? 1 : 0);
  }

  if (result.findings.length === 0) {
    console.log("No staleness detected.");
    return;
  }

  for (const finding of result.findings) {
    const icon =
      finding.severity === "error"
        ? "✗"
        : finding.severity === "warn"
          ? "⚠"
          : "ℹ";
    console.log(`${icon} ${finding.message}`);
  }

  process.exit(result.stale ? 1 : 0);
}

export function contextCommand(): Command {
  const cmd = new Command("context")
    .description("Show x-harness canonical context and refresh AGENTS.md")
    .option("--verbose", "Show full verbose context", false)
    .option("--json", "Output context as JSON")
    .option("--refresh", "Refresh AGENTS.md managed context block", false)
    .option("--check", "Validate managed block and linked files", false)
    .option("--contract", "Output generated canonical runtime contract", false)
    .option(
      "--write-contract-assets",
      "Refresh managed contract blocks in docs/templates/adapters",
      false
    )
    .option("--root <path>", "Repository root", process.cwd());

  cmd
    .command("staleness")
    .description("Detect drift in managed context and linked files")
    .option("--output-json", "Output JSON")
    .option("--staleness-root <path>", "Repository root", process.cwd())
    .action(async (opts: { outputJson?: boolean; stalenessRoot?: string }) => {
      await contextStalenessAction({
        json: opts.outputJson,
        root: opts.stalenessRoot,
      });
    });

  const manifestCmd = new Command("manifest").description(
    "Generate and check context manifest files"
  );

  manifestCmd
    .command("write")
    .description("Write a context manifest for the given files")
    .requiredOption("--files <paths>", "Comma-separated file paths")
    .option(
      "--out <path>",
      "Output manifest path",
      ".x-harness/context-manifest.yaml"
    )
    .option("--json", "Output JSON")
    .option("--reason <reason>", "Reason for the manifest")
    .action(
      async (
        opts: { files: string; out: string; json?: boolean; reason?: string },
        command: Command
      ) => {
        const files = opts.files
          .split(",")
          .map((f) => f.trim())
          .filter((f) => f !== "");
        if (files.length === 0) {
          console.error("Error: --files is required");
          process.exit(2);
        }
        const manifest = generateManifest(
          files,
          process.cwd(),
          opts.reason ?? ""
        );
        writeManifest(manifest, opts.out);
        const parentOpts = command.parent?.parent?.opts() as
          | Record<string, unknown>
          | undefined;
        const json =
          opts.json ?? (parentOpts?.json as boolean | undefined) ?? false;
        if (json) {
          console.log(
            JSON.stringify(
              {
                ok: true,
                out: opts.out,
                entries: manifest.entries.map((e) => ({
                  path: e.path,
                  sha256: e.sha256,
                })),
              },
              null,
              2
            )
          );
        } else {
          console.log(
            `wrote manifest (${manifest.entries.length} entries) to ${opts.out}`
          );
        }
      }
    );

  manifestCmd
    .command("check")
    .description("Check a context manifest for stale entries")
    .requiredOption("--manifest <path>", "Path to manifest file")
    .option("--json", "Output JSON")
    .action(
      async (opts: { manifest: string; json?: boolean }, command: Command) => {
        try {
          const manifest = readManifest(opts.manifest);
          validateManifest(manifest);
          const stale = checkManifest(manifest, process.cwd());
          const parentOpts = command.parent?.parent?.opts() as
            | Record<string, unknown>
            | undefined;
          const json =
            opts.json ?? (parentOpts?.json as boolean | undefined) ?? false;
          if (json) {
            console.log(
              JSON.stringify({ ok: stale.length === 0, stale }, null, 2)
            );
          } else {
            if (stale.length === 0) {
              console.log("manifest check passed: all entries fresh");
            } else {
              console.log(
                `manifest check failed: stale entries: ${stale.join(", ")}`
              );
            }
          }
          process.exitCode = stale.length > 0 ? 1 : 0;
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);
          const parentOpts = command.parent?.parent?.opts() as
            | Record<string, unknown>
            | undefined;
          const json =
            opts.json ?? (parentOpts?.json as boolean | undefined) ?? false;
          if (json) {
            console.log(JSON.stringify({ ok: false, error: message }));
          } else {
            console.error(`Error: ${message}`);
          }
          process.exitCode = 1;
        }
      }
    );

  cmd.addCommand(manifestCmd);

  cmd.action(async (opts: ContextOptions) => {
    const root = path.resolve(opts.root ?? process.cwd());

    if (opts.check) {
      return contextCheckAction(opts);
    }

    if (opts.refresh) {
      const agentsPath = path.join(root, "AGENTS.md");
      if (!(await fs.pathExists(agentsPath))) {
        console.error(`Error: AGENTS.md not found at ${agentsPath}`);
        process.exit(2);
      }
      const agentsContent = await fs.readFile(agentsPath, "utf-8");
      const block = generateManagedBlock();
      const updated = injectManagedBlock(agentsContent, block);
      await fs.writeFile(agentsPath, updated, "utf-8");
      const hashMatch = block.match(/<!-- context-hash: ([a-f0-9]+) -->/);
      const hash = hashMatch ? hashMatch[1] : "unknown";
      console.log(`AGENTS.md refreshed (context-hash: ${hash})`);
      return;
    }

    if (opts.writeContractAssets) {
      const contract = await loadRuntimeContract(root);
      const written: string[] = [];
      for (const target of MANAGED_CONTRACT_TARGETS) {
        const targetPath = path.join(root, target.path);
        if (!(await fs.pathExists(targetPath))) {
          console.error(`Error: contract target not found at ${targetPath}`);
          process.exit(2);
        }
        const content = await fs.readFile(targetPath, "utf-8");
        const block = generateManagedContractBlock(target, contract);
        const updated = injectManagedContractBlock(content, target, block);
        await fs.writeFile(targetPath, updated, "utf-8");
        written.push(target.path);
      }
      if (opts.json) {
        console.log(JSON.stringify({ written }, null, 2));
      } else {
        console.log(`Managed contract blocks refreshed: ${written.join(", ")}`);
      }
      return;
    }

    if (opts.contract) {
      const contract = await loadRuntimeContract(root);
      const markdown = renderRuntimeContractMarkdown(contract);
      const hash = getContextHash(markdown);

      if (opts.json) {
        console.log(
          JSON.stringify(
            {
              contract,
              markdown,
              hash,
            },
            null,
            2
          )
        );
        return;
      }

      console.log(markdown);
      console.log("");
      console.log(`contract-hash: ${hash}`);
      return;
    }

    const context = getCanonicalContext(opts.verbose);
    const hash = getContextHash(context);

    if (opts.json) {
      const agentsPath = path.join(root, "AGENTS.md");
      const agentsFresh = (await fs.pathExists(agentsPath))
        ? validateManagedBlock(await fs.readFile(agentsPath, "utf-8"))
        : { valid: false, note: "AGENTS.md not found" };

      console.log(
        JSON.stringify(
          {
            context,
            hash,
            mode: opts.verbose ? "verbose" : "compact",
            agents_fresh: agentsFresh.valid,
            agents_note: agentsFresh.note,
          },
          null,
          2
        )
      );
      return;
    }

    console.log(context);
    if (opts.verbose) {
      console.log("");
      console.log(`context-hash: ${hash}`);
    }
  });

  return cmd;
}
