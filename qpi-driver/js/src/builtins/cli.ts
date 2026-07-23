#!/usr/bin/env node
/**
 * The `qpi-driver` CLI that runs QPI's officially maintained TypeScript built-in
 * drivers, mirroring the Python `qpi-driver` CLI (RFC 0001 §4): the operation is
 * the subcommand (process | monitor), the device selects the backend within it,
 * universal flags are shared, and a device's own settings are passed as
 * repeatable `-o key=value`.
 *
 * Installed as the package's `qpi-driver` bin, so:
 *
 *   npm install -g qpi-driver   # or: npx -y qpi-driver …
 *   qpi-driver monitor --device bluefors_gen1 \
 *     --qpi-addr https://qpi.example.com --token … --ca-fingerprint … \
 *     -o base_url=http://localhost:49099 -o channels=mapper.bf.tmc:K
 */

import { Command } from "commander";

import type { QpiDriver } from "../driver.js";
import { BlueforsGen1Driver, parseChannels } from "./bluefors-gen1.js";

const VERSION = "0.1.0";

/** The universal options every operation subcommand shares (commander camelCases them). */
interface CommonOpts {
  qpiAddr: string;
  token: string;
  name: string;
  device: string;
  caFile: string;
  caFingerprint?: string;
  option: string[];
  recvTimeoutMs: number;
}

/** Builds (but does not run) the driver for one device from the parsed flags/options. */
type DeviceRunner = (
  common: CommonOpts,
  opts: Record<string, string>,
) => QpiDriver;

// TypeScript ships no process (QPU) built-in yet, so that registry is empty;
// both operations are dispatched the same way, so a new device is one entry.
const processDrivers: Record<string, DeviceRunner> = {};
const monitorDrivers: Record<string, DeviceRunner> = {
  bluefors_gen1: buildBlueforsGen1,
};

/**
 * Builds the Bluefors Gen. 1 monitor from the -o options. Recognised keys
 * mirror the Python and Go drivers: channels (required), base_url, api_key,
 * poll_interval (seconds), timeout (seconds).
 */
function buildBlueforsGen1(
  common: CommonOpts,
  opts: Record<string, string>,
): QpiDriver {
  const channels = opts.channels;
  if (!channels) {
    throw new Error(
      "bluefors_gen1 needs a 'channels' option, e.g. " +
        "-o channels=mapper.bf.tmc:K,mapper.bf.pmc:mbar",
    );
  }
  return new BlueforsGen1Driver({
    qpiAddr: common.qpiAddr,
    token: common.token,
    name: common.name,
    caFingerprint: common.caFingerprint,
    blueforsBaseUrl: opts.base_url,
    channels: parseChannels(channels),
    apiKey: opts.api_key,
    pollIntervalMs: opts.poll_interval
      ? Number(opts.poll_interval) * 1000
      : undefined,
    timeoutMs: opts.timeout ? Number(opts.timeout) * 1000 : undefined,
  });
}

async function runOperation(
  operation: string,
  registry: Record<string, DeviceRunner>,
  common: CommonOpts,
): Promise<void> {
  if (!common.token) {
    fail(
      "access token is required; set --token/-t or the QPI_ACCESS_TOKEN environment variable",
    );
  }
  const runner = registry[common.device];
  if (!runner) {
    const known = Object.keys(registry).sort().join(", ");
    fail(
      `unknown ${operation} device '${common.device}'; known devices: ${known}`,
    );
  }
  let driver: QpiDriver;
  try {
    driver = runner(common, parseOptions(common.option));
  } catch (err) {
    fail((err as Error).message);
  }
  await driver.run();
}

/** Turns repeatable `-o key=value` flags into a dict. */
function parseOptions(pairs: string[]): Record<string, string> {
  const opts: Record<string, string> = {};
  for (const pair of pairs) {
    const eq = pair.indexOf("=");
    const key = eq >= 0 ? pair.slice(0, eq).trim() : "";
    if (!key) {
      throw new Error(`invalid option '${pair}'; expected key=value`);
    }
    opts[key] = pair.slice(eq + 1).trim();
  }
  return opts;
}

function fail(message: string): never {
  console.error(`Error: ${message}`);
  process.exit(1);
}

function collect(value: string, previous: string[]): string[] {
  return previous.concat([value]);
}

function envOr(key: string, fallback: string): string {
  return process.env[key] || fallback;
}

function addOperation(
  program: Command,
  name: string,
  defaultDevice: string,
  defaultName: string,
  description: string,
  registry: Record<string, DeviceRunner>,
): void {
  program
    .command(name)
    .description(description)
    .option(
      "-a, --qpi-addr <url>",
      "Full URL of the QPI server",
      envOr("QPI_ADDR", "http://127.0.0.1:8090"),
    )
    .option(
      "-t, --token <token>",
      "Access token identifying this driver",
      process.env.QPI_ACCESS_TOKEN || "",
    )
    .option(
      "-n, --name <name>",
      "Human-readable name for this driver",
      envOr("QPI_DRIVER_NAME", defaultName),
    )
    .option(
      "-d, --device <device>",
      "Which backend to run within the operation",
      envOr("QPI_DEVICE", defaultDevice),
    )
    .option(
      "--ca-file <path>",
      "Where the downloaded server root CA is written",
      envOr("QPI_CA_FILE", "./bin/qpi.ca.pem"),
    )
    .option(
      "--ca-fingerprint <hex>",
      "SHA-256 fingerprint pinning the downloaded root CA",
      process.env.QPI_CA_FINGERPRINT,
    )
    .option(
      "-o, --option <keyvalue>",
      "Operation config as key=value, repeatable",
      collect,
      [],
    )
    .option(
      "--recv-timeout-ms <ms>",
      "Receive loop timeout in ms",
      (v) => parseInt(v, 10),
      Number(process.env.QPI_RECV_TIMEOUT_MS) || 200,
    )
    .action((opts: CommonOpts) => runOperation(name, registry, opts));
}

const program = new Command();
program
  .name("qpi-driver")
  .description("Quantum Processing Interface (QPI) Driver CLI")
  .version(VERSION);

addOperation(
  program,
  "process",
  "mock",
  "qpu_sim_01",
  "Run a process driver — a QPU that executes jobs pushed to it (RFC 0001 §4).",
  processDrivers,
);
addOperation(
  program,
  "monitor",
  "bluefors_gen1",
  "qpi-monitor",
  "Run a monitor driver — one that only reports upward on its own schedule (RFC 0001 §7).",
  monitorDrivers,
);

program
  .command("version")
  .description("Show the version of the QPI driver CLI")
  .action(() => console.log(VERSION));

program.parseAsync(process.argv).catch((err) => {
  console.error("Error:", err);
  process.exit(1);
});
