#!/usr/bin/env node
// Runs the burrow-g binary from the platform package npm installed for this machine.
const { spawnSync } = require("child_process");

const packages = {
  "linux x64": "burrow-g-linux-x64",
  "linux arm64": "burrow-g-linux-arm64",
  "darwin x64": "burrow-g-darwin-x64",
  "darwin arm64": "burrow-g-darwin-arm64",
  "win32 x64": "burrow-g-win32-x64",
  "win32 arm64": "burrow-g-win32-arm64",
};

const pkg = packages[`${process.platform} ${process.arch}`];
if (!pkg) {
  console.error(`burrow-g: ${process.platform}/${process.arch} is not supported yet`);
  process.exit(1);
}

let binary;
try {
  const name = process.platform === "win32" ? "burrow-g.exe" : "burrow-g";
  binary = require.resolve(`${pkg}/bin/${name}`);
} catch {
  console.error(`burrow-g: ${pkg} is missing. Reinstall with: npm install -g burrow-g`);
  process.exit(1);
}

// The binary (and any agent it resumes) handles ctrl+c itself.
process.on("SIGINT", () => {});
process.on("SIGTERM", () => {});

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error(`burrow-g: ${result.error.message}`);
  process.exit(1);
}
process.exit(result.status ?? 1);
