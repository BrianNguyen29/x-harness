#!/usr/bin/env node
import { Command } from "commander";
import { initCommand } from "./commands/init.js";
import { addCommand } from "./commands/add.js";
import { verifyCommand } from "./commands/verify.js";
import { doctorCommand } from "./commands/doctor.js";
import { reportCommand } from "./commands/report.js";
import { traceCommand } from "./commands/trace.js";
import { handoffCommand } from "./commands/handoff.js";

const program = new Command();
program.name("x-harness").description("A lightweight verify-gated harness for AI-agent workflows").version("0.1.0");
program.addCommand(initCommand());
program.addCommand(addCommand());
program.addCommand(handoffCommand());
program.addCommand(verifyCommand());
program.addCommand(traceCommand());
program.addCommand(reportCommand());
program.addCommand(doctorCommand());
program.parse(process.argv);
