"use strict";

const version = process.versions.node || "";
const match = /^(\d+)\.(\d+)\.(\d+)$/.exec(version);

if (!match) {
  console.error(`[ossam] Unable to parse Node.js version: ${version}`);
  process.exit(1);
}

const major = Number(match[1]);
const minor = Number(match[2]);

const isSupported = major === 24 && minor === 14;

if (!isSupported) {
  console.error(
    `[ossam] Unsupported Node.js version: ${version}. Required: >=24.14.0 <24.15.0 (Node 24.14.x).`
  );
  process.exit(1);
}
