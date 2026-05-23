/**
 * Centralized CLI exit handling.
 * Provides a typed error class and exit helper to avoid scattered process.exit calls.
 */

export class CliError extends Error {
  constructor(
    message: string,
    public readonly exitCode: number = 1
  ) {
    super(message);
    this.name = "CliError";
  }
}

export function exitWithError(message: string, exitCode = 1): never {
  console.error(`x-harness error: ${message}`);
  process.exit(exitCode);
}

export function handleCliError(error: unknown): never {
  if (error instanceof CliError) {
    console.error(`x-harness error: ${error.message}`);
    process.exit(error.exitCode);
  }
  const message = error instanceof Error ? error.message : String(error);
  console.error(`x-harness error: ${message}`);
  process.exit(1);
}
