#!/usr/bin/env node
import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  ".."
);
const nodeEntrypoint = path.join(packageRoot, "dist", "index.js");

const platformMap = new Map([
  ["linux", "linux"],
  ["darwin", "darwin"],
  ["win32", "windows"],
]);

const archMap = new Map([
  ["x64", "amd64"],
  ["arm64", "arm64"],
]);

async function packageVersion() {
  try {
    const raw = await readFile(path.join(packageRoot, "package.json"), "utf8");
    return JSON.parse(raw).version ?? "0.0.0";
  } catch {
    return "0.0.0";
  }
}

async function candidateGoBinaries() {
  const goos = platformMap.get(process.platform);
  const goarch = archMap.get(process.arch);
  if (!goos || !goarch) return [];

  const version = await packageVersion();
  const ext = goos === "windows" ? ".exe" : "";
  return [
    path.join(
      packageRoot,
      "go-binaries",
      `x-harness-v${version}-${goos}-${goarch}${ext}`
    ),
    path.join(packageRoot, "go-binaries", `x-harness-${goos}-${goarch}${ext}`),
    path.join(packageRoot, "go-binaries", `x-harness${ext}`),
  ];
}

function run(command, args) {
  const child = spawn(command, args, { stdio: "inherit" });
  child.on("error", () => {
    process.exit(1);
  });
  child.on("exit", (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
      return;
    }
    process.exit(code ?? 1);
  });
}

function runNodeFallback() {
  run(process.execPath, [nodeEntrypoint, ...process.argv.slice(2)]);
}

async function main() {
  const nodeEntrypointExists = existsSync(nodeEntrypoint);

  if (process.env.X_HARNESS_GO === "0") {
    if (nodeEntrypointExists) {
      runNodeFallback();
      return;
    }
    console.error(
      "x-harness: X_HARNESS_GO=0 requested but Node fallback is not available in this installation."
    );
    process.exit(1);
  }

  for (const candidate of await candidateGoBinaries()) {
    if (existsSync(candidate)) {
      run(candidate, process.argv.slice(2));
      return;
    }
  }

  if (nodeEntrypointExists) {
    runNodeFallback();
    return;
  }

  console.error(
    "x-harness: No Go binary found for your platform and Node fallback is not available in this installation."
  );
  process.exit(1);
}

main().catch(() => process.exit(1));
