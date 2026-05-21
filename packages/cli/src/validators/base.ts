import { loadSchema, compileSchema } from "../core/schema.js";

export interface ValidationResult {
  valid: boolean;
  errors: string[];
}

export async function validateAgainstSchema(data: unknown, schemaName: string): Promise<ValidationResult> {
  try {
    const schema = await loadSchema(schemaName);
    const validate = compileSchema(schema);
    const valid = validate(data) as boolean;
    if (valid) {
      return { valid: true, errors: [] };
    }
    const errors = validate.errors?.map((e) => `${e.instancePath || "/"} ${e.message}`) ?? ["validation failed"];
    return { valid: false, errors };
  } catch (err) {
    return { valid: false, errors: [`schema load error: ${err instanceof Error ? err.message : String(err)}`] };
  }
}
