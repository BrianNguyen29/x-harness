import { createHash } from "node:crypto";
import fs from "fs-extra";

export async function sha256File(filePath: string): Promise<string | null> {
  if (!(await fs.pathExists(filePath))) return null;
  const content = await fs.readFile(filePath, "utf-8");
  return sha256String(content);
}

export function sha256String(input: string): string {
  return createHash("sha256").update(input, "utf-8").digest("hex");
}
