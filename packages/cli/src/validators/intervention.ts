import { validateAgainstSchema, type ValidationResult } from "./base.js";

export async function validate(data: unknown): Promise<ValidationResult> {
  const schemaResult = await validateAgainstSchema(data, "intervention");
  if (!schemaResult.valid) return schemaResult;

  const artifact = data as { authorizer?: unknown };
  if (
    typeof artifact.authorizer !== "string" ||
    artifact.authorizer.trim() === ""
  ) {
    return {
      valid: false,
      errors: ["/authorizer must be a non-empty string"],
    };
  }

  return schemaResult;
}
