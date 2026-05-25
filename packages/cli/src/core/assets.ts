import * as path from "node:path";
import { fileURLToPath } from "node:url";
import fs from "fs-extra";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const ASSET_MARKERS = [
  "templates/COMPLETION_CARD.md",
  "policies/admission.yaml",
  "schemas/completion-card.schema.json",
];

async function hasAssetMarkers(root: string): Promise<boolean> {
  for (const marker of ASSET_MARKERS) {
    if (!(await fs.pathExists(path.join(root, marker)))) {
      return false;
    }
  }
  return true;
}

async function findSourceCheckoutRoot(start: string): Promise<string | null> {
  let current = path.resolve(start);
  while (true) {
    if (
      (await fs.pathExists(path.join(current, "X_HARNESS.md"))) &&
      (await fs.pathExists(path.join(current, "packages", "cli")))
    ) {
      return current;
    }
    const parent = path.dirname(current);
    if (parent === current) return null;
    current = parent;
  }
}

export async function resolveAssetRoot(): Promise<string> {
  const envRoot = process.env.X_HARNESS_ASSET_ROOT;
  if (envRoot && (await hasAssetMarkers(envRoot))) {
    return path.resolve(envRoot);
  }

  const sourceRoot = await findSourceCheckoutRoot(__dirname);
  if (sourceRoot && (await hasAssetMarkers(sourceRoot))) {
    return sourceRoot;
  }

  const packageRoot = path.resolve(__dirname, "..", "..");
  if (await hasAssetMarkers(packageRoot)) {
    return packageRoot;
  }

  const cwdSourceRoot = await findSourceCheckoutRoot(process.cwd());
  if (cwdSourceRoot && (await hasAssetMarkers(cwdSourceRoot))) {
    return cwdSourceRoot;
  }

  throw new Error(
    "x-harness package assets not found; run npm run build in a source checkout or reinstall the package"
  );
}

export async function resolveAssetPath(relativePath: string): Promise<string> {
  const root = await resolveAssetRoot();
  return path.join(root, relativePath);
}
