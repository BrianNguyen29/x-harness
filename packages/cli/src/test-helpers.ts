import { execFile } from "node:child_process";
import * as path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export function execaNode(
  args: string[]
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  return new Promise((resolve) => {
    const script = path.join(__dirname, "..", "dist", "index.js");
    execFile(
      "node",
      [script, ...args],
      { cwd: path.join(__dirname, "..") },
      (error, stdout, stderr) => {
        resolve({
          stdout: stdout.trim(),
          stderr: stderr.trim(),
          exitCode: error?.code ? Number(error.code) : 0,
        });
      }
    );
  });
}
