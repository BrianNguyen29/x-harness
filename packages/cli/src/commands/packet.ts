import { Command } from "commander";
import * as path from "node:path";
import fs from "fs-extra";
import {
  createPacket,
  listPacketsForTask,
  verifyPacketChain,
} from "../core/packet.js";

export function packetCommand(): Command {
  const cmd = new Command("packet").description(
    "Create and verify claim packets"
  );

  cmd
    .command("create")
    .description("Create a claim packet from a completion card")
    .option(
      "--card <path>",
      "Path to completion card YAML",
      "completion-card.yaml"
    )
    .option("--packets-dir <dir>", "Packets directory", ".x-harness/packets")
    .action(async (opts: { card?: string; packetsDir?: string }) => {
      const cardPath = path.resolve(opts.card ?? "completion-card.yaml");
      if (!(await fs.pathExists(cardPath))) {
        console.error(`Error: Completion card not found at ${cardPath}`);
        process.exit(2);
      }

      try {
        const { packet, filePath } = await createPacket(
          cardPath,
          opts.packetsDir
        );
        console.log(`packet created: ${filePath}`);
        console.log(`packet_id: ${packet.packet_id}`);
        console.log(`task_id: ${packet.task_id}`);
        console.log(`previous_packet_id: ${packet.previous_packet_id}`);
        console.log(`payload_hash: ${packet.payload_hash}`);
      } catch (err) {
        console.error(
          `Error creating packet: ${err instanceof Error ? err.message : String(err)}`
        );
        process.exit(1);
      }
    });

  cmd
    .command("verify-chain")
    .description("Verify the integrity of a packet chain for a task")
    .option("--task-id <id>", "Task ID", "")
    .option("--packets-dir <dir>", "Packets directory", ".x-harness/packets")
    .action(async (opts: { taskId?: string; packetsDir?: string }) => {
      const taskId = opts.taskId ?? "";
      if (!taskId) {
        console.error("Error: --task-id is required");
        process.exit(2);
      }

      const packetsDir = path.resolve(opts.packetsDir ?? ".x-harness/packets");
      if (!(await fs.pathExists(packetsDir))) {
        console.error(`Error: Packets directory not found: ${packetsDir}`);
        process.exit(2);
      }

      const packets = await listPacketsForTask(taskId, packetsDir);
      if (packets.length === 0) {
        console.error(`Error: No packets found for task ${taskId}`);
        process.exit(2);
      }

      const result = verifyPacketChain(packets);

      if (result.valid) {
        console.log(`chain valid: ${result.packetsChecked} packet(s) checked`);
        process.exit(0);
      } else {
        console.error(`chain broken: ${result.reason}`);
        if (result.firstBrokenPacketId) {
          console.error(`packet: ${result.firstBrokenPacketId}`);
        }
        if (result.expectedHash && result.actualHash) {
          console.error(`expected hash: ${result.expectedHash}`);
          console.error(`actual hash:   ${result.actualHash}`);
        }
        process.exit(1);
      }
    });

  return cmd;
}
