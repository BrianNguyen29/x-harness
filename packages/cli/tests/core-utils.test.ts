import { describe, it, expect } from "vitest";
import * as path from "node:path";
import * as fs from "node:fs";
import * as os from "node:os";

// Core utilities to test
import {
  readYamlOrJson,
  readJsonl,
  loadSchema,
  compileSchema,
} from "../src/core/schema.js";
import { sha256File, sha256String } from "../src/core/hash.js";
import { validateAgainstSchema } from "../src/validators/base.js";

describe("schema utilities", () => {
  describe("readYamlOrJson", () => {
    it("reads valid YAML content", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-schema-"));
      const yamlPath = path.join(tmpDir, "test.yaml");
      fs.writeFileSync(yamlPath, "key: value\nlist:\n  - a\n  - b");
      try {
        const result = await readYamlOrJson(yamlPath);
        expect(result).toEqual({ key: "value", list: ["a", "b"] });
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("reads valid JSON content", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-schema-"));
      const jsonPath = path.join(tmpDir, "test.json");
      fs.writeFileSync(jsonPath, '{"key": "value", "list": [1, 2]}');
      try {
        const result = await readYamlOrJson(jsonPath);
        expect(result).toEqual({ key: "value", list: [1, 2] });
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("falls back to YAML when extension is unknown", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-schema-"));
      const txtPath = path.join(tmpDir, "test.txt");
      fs.writeFileSync(txtPath, "key: value");
      try {
        const result = await readYamlOrJson(txtPath);
        expect(result).toEqual({ key: "value" });
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("throws on malformed YAML", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-schema-"));
      const yamlPath = path.join(tmpDir, "test.yaml");
      fs.writeFileSync(yamlPath, "key: [unclosed");
      try {
        await expect(readYamlOrJson(yamlPath)).rejects.toThrow();
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("throws on malformed JSON", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-schema-"));
      const jsonPath = path.join(tmpDir, "test.json");
      fs.writeFileSync(jsonPath, '{"key": invalid}');
      try {
        await expect(readYamlOrJson(jsonPath)).rejects.toThrow();
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });
  });

  describe("readJsonl", () => {
    it("reads valid JSONL content", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-schema-"));
      const jsonlPath = path.join(tmpDir, "test.jsonl");
      fs.writeFileSync(jsonlPath, '{"a": 1}\n{"b": 2}\n{"c": 3}');
      try {
        const result = await readJsonl(jsonlPath);
        expect(result).toEqual([{ a: 1 }, { b: 2 }, { c: 3 }]);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty array for non-existent file", async () => {
      const result = await readJsonl("/nonexistent/path.jsonl");
      expect(result).toEqual([]);
    });

    it("skips empty lines", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-schema-"));
      const jsonlPath = path.join(tmpDir, "test.jsonl");
      fs.writeFileSync(jsonlPath, '{"a": 1}\n\n{"b": 2}\n\n');
      try {
        const result = await readJsonl(jsonlPath);
        expect(result).toEqual([{ a: 1 }, { b: 2 }]);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });
  });

  describe("loadSchema", () => {
    it("loads existing schema by name", async () => {
      const schema = await loadSchema("claim");
      expect(schema).toBeDefined();
      expect(typeof schema).toBe("object");
    });

    it("throws for non-existent schema", async () => {
      await expect(loadSchema("nonexistent-schema")).rejects.toThrow(
        "Schema not found"
      );
    });
  });

  describe("compileSchema", () => {
    it("compiles a valid schema", () => {
      const schema = {
        type: "object",
        properties: {
          name: { type: "string" },
        },
        required: ["name"],
      };
      const validate = compileSchema(schema);
      expect(typeof validate).toBe("function");
      expect(validate({ name: "test" })).toBe(true);
      expect(validate({})).toBe(false);
    });
  });
});

describe("hash utilities", () => {
  describe("sha256String", () => {
    it("produces consistent hash for same input", () => {
      const input = "hello world";
      const hash1 = sha256String(input);
      const hash2 = sha256String(input);
      expect(hash1).toBe(hash2);
    });

    it("produces different hash for different inputs", () => {
      const hash1 = sha256String("hello");
      const hash2 = sha256String("world");
      expect(hash1).not.toBe(hash2);
    });

    it("produces 64-character hex string (sha256)", () => {
      const hash = sha256String("test");
      expect(hash).toMatch(/^[a-f0-9]{64}$/);
    });

    it("produces empty string hash for empty input", () => {
      const hash = sha256String("");
      expect(hash).toMatch(/^[a-f0-9]{64}$/);
    });
  });

  describe("sha256File", () => {
    it("returns hash for existing file", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "xh-hash-"));
      const filePath = path.join(tmpDir, "test.txt");
      fs.writeFileSync(filePath, "file content");
      try {
        const hash = await sha256File(filePath);
        expect(hash).toMatch(/^[a-f0-9]{64}$/);
        // Verify it matches direct string hash
        const expectedHash = sha256String("file content");
        expect(hash).toBe(expectedHash);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns null for non-existent file", async () => {
      const hash = await sha256File("/nonexistent/file.txt");
      expect(hash).toBeNull();
    });
  });
});

describe("validateAgainstSchema (base validator)", () => {
  describe("happy path", () => {
    it("validates valid claim data against claim schema", async () => {
      const data = { fix_status: "fixed" };
      const result = await validateAgainstSchema(data, "claim");
      expect(result.valid).toBe(true);
      expect(result.errors).toEqual([]);
    });

    it("validates valid evidence data against evidence schema", async () => {
      const data = { files_changed: ["src/index.ts"] };
      const result = await validateAgainstSchema(data, "evidence");
      expect(result.valid).toBe(true);
      expect(result.errors).toEqual([]);
    });
  });

  describe("error path", () => {
    it("returns error for invalid data", async () => {
      const data = { fix_status: "invalid-status" };
      const result = await validateAgainstSchema(data, "claim");
      expect(result.valid).toBe(false);
      expect(result.errors.length).toBeGreaterThan(0);
    });

    it("returns error for missing required fields", async () => {
      const data = {};
      const result = await validateAgainstSchema(data, "claim");
      expect(result.valid).toBe(false);
      expect(result.errors.length).toBeGreaterThan(0);
    });

    it("returns error for non-existent schema", async () => {
      const data = { key: "value" };
      const result = await validateAgainstSchema(data, "nonexistent-schema");
      expect(result.valid).toBe(false);
      expect(result.errors.some((e) => e.includes("Schema not found"))).toBe(
        true
      );
    });
  });
});
