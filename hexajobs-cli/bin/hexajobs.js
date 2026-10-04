#!/usr/bin/env node
'use strict';
// Wrapper for the hexajobs native binary.
// Resolves the binary in this order:
//   1. $HEXAJOBS_BIN (explicit override)
//   2. <package>/binaries/hexajobs (downloaded by postinstall)
//   3. <package>/bin/hexajobs-linux-<arch> (local `make build-linux-*`)
//   4. <package>/bin/hexajobs (local `make build`)
const { spawnSync } = require('node:child_process');
const { existsSync } = require('node:fs');
const { join } = require('node:path');

function goArch() {
  if (process.arch === 'x64') return 'amd64';
  if (process.arch === 'arm64') return 'arm64';
  return null;
}

function resolveBinary() {
  if (process.env.HEXAJOBS_BIN && existsSync(process.env.HEXAJOBS_BIN)) {
    return process.env.HEXAJOBS_BIN;
  }
  const pkgDir = join(__dirname, '..');
  const arch = goArch();
  const candidates = [join(pkgDir, 'binaries', 'hexajobs')];
  if (arch) {
    candidates.push(join(pkgDir, 'bin', `hexajobs-linux-${arch}`));
  }
  candidates.push(join(pkgDir, 'bin', 'hexajobs'));
  return candidates.find((p) => existsSync(p)) || null;
}

function main() {
  const bin = resolveBinary();
  if (!bin) {
    console.error(
      'hexajobs: native binary not found.\n' +
        'Reinstall the package (postinstall downloads it automatically), or:\n' +
        '  - build from source: make build-linux-amd64 && ./scripts/install-linux.sh\n' +
        '  - point HEXAJOBS_BIN to your binary, e.g. HEXAJOBS_BIN=/usr/bin/hexajobs hexajobs --demo'
    );
    process.exit(1);
  }
  const res = spawnSync(bin, process.argv.slice(2), { stdio: 'inherit' });
  if (res.error) {
    console.error(`hexajobs: failed to launch binary: ${res.error.message}`);
    process.exit(1);
  }
  process.exit(res.status === null ? 1 : res.status);
}

main();
