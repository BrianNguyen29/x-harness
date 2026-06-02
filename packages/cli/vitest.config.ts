import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    testTimeout: 15000,
    // Several test files create scratch files under the package working
    // tree (e.g. trace-hash, packet, add command fixtures). The verify
    // command's mutation guard inspects the git working tree, so running
    // test files in parallel causes the verify command in one file to
    // observe a sibling test file's scratch files as unexpected changes
    // and produce a flaky `verifier_not_read_only` blocking predicate.
    // Serialize file execution to keep the working tree clean for each
    // file's verify runs.
    fileParallelism: false,
  },
});
