import { describe, it, expect } from "vitest";
import { execaNode } from "../src/test-helpers.js";

describe("progressive disclosure", () => {
  it("no args shows start-here guide", async () => {
    const { stdout, exitCode } = await execaNode([]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Start here");
    expect(stdout).toContain("check");
    expect(stdout).toContain("init");
    expect(stdout).toContain("--help-all");
    expect(stdout).toContain("Getting started");
    expect(stdout).not.toContain("Daily tasks");
    expect(stdout).not.toContain("Health & recovery");
    expect(stdout).not.toContain("Automation");
  });

  it("default help shows only core onboarding commands", async () => {
    const { stdout, exitCode } = await execaNode(["--help"]);
    expect(exitCode).toBe(0);
    // Core onboarding commands should be present
    expect(stdout).toContain("init");
    expect(stdout).toContain("doctor");
    expect(stdout).toContain("verify");
    expect(stdout).toContain("check");
    // Beta, advanced, and experimental commands should not appear
    expect(stdout).not.toContain("\n  xh start");
    expect(stdout).not.toContain("actions");
    expect(stdout).not.toContain("prepare");
    expect(stdout).not.toContain("packet");
    expect(stdout).not.toContain("intake");
    expect(stdout).not.toContain("governance");
    expect(stdout).not.toContain("federation");
    // Only the core onboarding category should be present
    expect(stdout).toContain("Getting started");
    expect(stdout).not.toContain("Daily tasks");
    expect(stdout).not.toContain("Health & recovery");
    expect(stdout).not.toContain("Automation");
    // Footer should be present
    expect(stdout).toContain("--help-all");
    expect(stdout).toContain("--help-maturity");
  });

  it("--help-all shows advanced commands", async () => {
    const { stdout, exitCode } = await execaNode(["--help-all"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("trace");
    expect(stdout).toContain("benchmark");
    expect(stdout).toContain("packet");
    expect(stdout).toContain("intake");
    expect(stdout).toContain("check");
    expect(stdout).toContain("doctor");
  });

  it("--help-maturity groups commands by maturity", async () => {
    const { stdout, exitCode } = await execaNode(["--help-maturity"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("stable:");
    expect(stdout).toContain("beta:");
    expect(stdout).toContain("experimental:");
    expect(stdout).toContain("check");
    expect(stdout).toContain("packet");
    expect(stdout).toContain("intake");
  });

  it("no args --lang vi shows Vietnamese start-here guide", async () => {
    const { stdout, exitCode } = await execaNode(["--lang", "vi"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Bắt đầu");
    expect(stdout).not.toContain("Tác vụ hằng ngày");
    expect(stdout).not.toContain("Sức khỏe & khôi phục");
    expect(stdout).not.toContain("Tự động hóa");
    expect(stdout).toContain("Các lệnh thường dùng và cách sử dụng");
    expect(stdout).toContain("Tất cả lệnh");
    expect(stdout).toContain("Các lệnh theo nhóm độ ổn định");
  });

  it("--help --lang vi shows Vietnamese help", async () => {
    const { stdout, exitCode } = await execaNode(["--help", "--lang", "vi"]);
    expect(exitCode).toBe(0);
    expect(stdout).toContain("Bắt đầu");
    expect(stdout).toContain("Cách dùng:");
  });

  it("advanced commands still execute when called directly", async () => {
    const { stdout, exitCode } = await execaNode(["doctor"]);
    expect(exitCode).toBe(0);
    const output = JSON.parse(stdout);
    expect(output).toHaveProperty("healthy");
  });
});
