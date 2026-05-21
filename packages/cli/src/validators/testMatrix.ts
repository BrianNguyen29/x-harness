import { validateAgainstSchema, type ValidationResult } from "./base.js";

export async function validate(data: unknown): Promise<ValidationResult> {
  return validateAgainstSchema(data, "test-matrix");
}
