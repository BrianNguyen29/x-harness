import { execFile } from "node:child_process";

export type ChangedFilesSource = "card" | "git" | "union" | "strict";

export interface ChangedFilesResolution {
  source: ChangedFilesSource;
  card_files: string[];
  git_files: string[];
  files: string[];
  errors: string[];
  notes: string[];
}

function normalizeFilePath(filePath: string): string {
  return filePath.replace(/\\/g, "/").replace(/^\.\//, "").trim();
}

function uniqueNormalized(files: string[]): string[] {
  return [...new Set(files.map(normalizeFilePath).filter(Boolean))].sort();
}

export function parseChangedFilesSource(value?: string): ChangedFilesSource {
  const normalized = value?.trim() || "card";
  if (
    normalized === "card" ||
    normalized === "git" ||
    normalized === "union" ||
    normalized === "strict"
  ) {
    return normalized;
  }
  throw new Error(
    "--changed-files-source must be one of: card, git, union, strict"
  );
}

export function defaultChangedFilesSource(input: {
  explicit?: string;
  diffRef?: string;
  strict?: boolean;
}): ChangedFilesSource {
  if (input.explicit) return parseChangedFilesSource(input.explicit);
  if (input.diffRef && input.strict) return "strict";
  if (input.diffRef) return "union";
  return "card";
}

export async function getGitDiffFiles(
  diffRef: string,
  root: string
): Promise<string[]> {
  if (!diffRef || diffRef.trim().length === 0) return [];
  const output = await new Promise<string>((resolve, reject) => {
    execFile(
      "git",
      ["diff", "--name-only", diffRef],
      { cwd: root },
      (error, stdout, stderr) => {
        if (error) {
          reject(
            new Error(
              `git diff --name-only ${diffRef} failed: ${stderr.trim() || error.message}`
            )
          );
          return;
        }
        resolve(stdout);
      }
    );
  });
  return uniqueNormalized(output.split("\n"));
}

export async function resolveChangedFiles(input: {
  cardFiles: string[];
  diffRef?: string;
  root: string;
  source: ChangedFilesSource;
}): Promise<ChangedFilesResolution> {
  const cardFiles = uniqueNormalized(input.cardFiles);
  const errors: string[] = [];
  const notes: string[] = [];
  let gitFiles: string[] = [];

  if (input.source !== "card") {
    if (!input.diffRef) {
      errors.push(
        `changed-files-source "${input.source}" requires --diff <ref>`
      );
    } else {
      gitFiles = await getGitDiffFiles(input.diffRef, input.root);
      notes.push(
        `git diff changed-files source ${input.diffRef}: ${gitFiles.length} file(s)`
      );
    }
  }

  let files = cardFiles;
  if (input.source === "git") {
    files = gitFiles;
  } else if (input.source === "union" || input.source === "strict") {
    files = uniqueNormalized([...cardFiles, ...gitFiles]);
  }

  if (input.source === "strict") {
    const cardSet = new Set(cardFiles);
    const gitSet = new Set(gitFiles);
    const missingFromCard = gitFiles.filter((file) => !cardSet.has(file));
    const extraInCard = cardFiles.filter((file) => !gitSet.has(file));
    if (missingFromCard.length > 0) {
      errors.push(
        `evidence.files_changed missing git diff file(s): ${missingFromCard.join(", ")}`
      );
    }
    if (extraInCard.length > 0) {
      errors.push(
        `evidence.files_changed includes file(s) absent from git diff: ${extraInCard.join(", ")}`
      );
    }
  }

  return {
    source: input.source,
    card_files: cardFiles,
    git_files: gitFiles,
    files,
    errors,
    notes,
  };
}
