import { afterEach, describe, expect, it } from "vitest";
import fs from "fs-extra";
import * as os from "node:os";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import { execaNode } from "../src/test-helpers.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(path.join(__dirname, "..", "..", ".."));
const tempDirs: string[] = [];

function makeTempDir(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-attribution-"));
  tempDirs.push(dir);
  return dir;
}

async function createWithheldEpisode(episodesDir: string): Promise<string> {
  const cardPath = path.join(
    repoRoot,
    "examples",
    "golden",
    "capability",
    "failed-typecheck-recovery-route",
    "completion-card.yaml"
  );
  const { stdout, exitCode } = await execaNode([
    "verify",
    "--card",
    cardPath,
    "--episode",
    "--episodes-dir",
    episodesDir,
    "--json",
  ]);
  expect(exitCode).toBe(1);
  const output = JSON.parse(stdout) as { episode: { episode_dir: string } };
  return path.resolve(repoRoot, output.episode.episode_dir);
}

afterEach(async () => {
  for (const dir of tempDirs.splice(0)) {
    await fs.remove(dir);
  }
});

describe("attribution command", () => {
  it("writes failure attribution for withheld episodes", async () => {
    const episodesDir = path.join(makeTempDir(), "episodes");
    const episodeDir = await createWithheldEpisode(episodesDir);
    const attributionPath = path.join(episodeDir, "failure-attribution.json");

    expect(await fs.pathExists(attributionPath)).toBe(true);
    const attribution = await fs.readJson(attributionPath);
    expect(attribution.admission_authority).toBe(false);
    expect(attribution.verdict.acceptance_status).toBe("withheld");
    expect(attribution.primary.taxonomy).toBe("Fverification");
    expect(attribution.primary.component_id).toBe("admission_policy");
  });

  it("explains attribution for one episode", async () => {
    const episodesDir = path.join(makeTempDir(), "episodes");
    const episodeDir = await createWithheldEpisode(episodesDir);

    const { stdout, exitCode } = await execaNode([
      "attribution",
      "explain",
      "--episode",
      episodeDir,
      "--json",
    ]);

    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.primary.taxonomy).toBe("Fverification");
    expect(output.unknown_rate_signal.is_unknown).toBe(false);
  });

  it("reports repeated attribution predicates across episodes", async () => {
    const episodesDir = path.join(makeTempDir(), "episodes");
    await createWithheldEpisode(episodesDir);
    await createWithheldEpisode(episodesDir);

    const { stdout, exitCode } = await execaNode([
      "attribution",
      "report",
      "--episodes-dir",
      episodesDir,
      "--group-by",
      "predicate",
      "--json",
    ]);

    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output.withheld_episodes).toBe(2);
    expect(output.unknown_rate).toBe(0);
    expect(output.groups[0].count).toBe(2);
    expect(output.groups[0].taxonomies).toContain("Fverification");
  });
});
