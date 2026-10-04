'use strict';
// postinstall: fetch the prebuilt Linux binary from GitHub Releases,
// or build from source when Go + Make are available.
// Never fails the install: on failure it prints guidance and exits 0,
// the bin wrapper reports a clear error at runtime instead.
const { spawnSync } = require('node:child_process');
const { createWriteStream, existsSync, mkdirSync, chmodSync, copyFileSync } = require('node:fs');
const { get } = require('node:https');
const { join } = require('node:path');

const pkg = require('../package.json');

const REPO = 'hexadesign24/HexaJobs';

function goArch() {
  if (process.arch === 'x64') return 'amd64';
  if (process.arch === 'arm64') return 'arm64';
  return null;
}

function download(url, dest, redirects = 5) {
  return new Promise((resolve, reject) => {
    get(url, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        if (redirects === 0) {
          reject(new Error('too many redirects'));
          return;
        }
        res.resume();
        download(res.headers.location, dest, redirects - 1).then(resolve, reject);
        return;
      }
      if (res.statusCode !== 200) {
        res.resume();
        reject(new Error(`HTTP ${res.statusCode} for ${url}`));
        return;
      }
      const out = createWriteStream(dest, { mode: 0o755 });
      res.pipe(out);
      out.on('finish', () => resolve());
      out.on('error', reject);
    }).on('error', reject);
  });
}

function tryBuildFromSource(pkgDir, arch) {
  const go = spawnSync('go', ['version'], { stdio: 'ignore' });
  const make = spawnSync('make', ['--version'], { stdio: 'ignore' });
  if (go.error || make.error) return null;
  console.log('hexajobs: release download unavailable, building from source...');
  const build = spawnSync('make', [`build-linux-${arch}`], { cwd: pkgDir, stdio: 'inherit' });
  if (build.status !== 0) return null;
  return join(pkgDir, 'bin', `hexajobs-linux-${arch}`);
}

async function main() {
  if (process.env.HEXAJOBS_SKIP_DOWNLOAD) {
    console.log('hexajobs: HEXAJOBS_SKIP_DOWNLOAD set, skipping binary download.');
    return;
  }
  if (process.platform !== 'linux') {
    console.warn(
      `hexajobs: prebuilt binaries are currently Linux-only (detected ${process.platform}).\n` +
        'Build from source with Go 1.24+: go build -o bin/hexajobs ./cmd/hexajobs, ' +
        'then set HEXAJOBS_BIN to the binary path.'
    );
    return;
  }
  const arch = goArch();
  if (!arch) {
    console.warn(`hexajobs: unsupported architecture ${process.arch}; only x64 and arm64 are supported.`);
    return;
  }
  const pkgDir = join(__dirname, '..');
  const destDir = join(pkgDir, 'binaries');
  const dest = join(destDir, 'hexajobs');
  if (existsSync(dest)) return; // already installed
  mkdirSync(destDir, { recursive: true });

  const url = `https://github.com/${REPO}/releases/download/v${pkg.version}/hexajobs-linux-${arch}`;
  try {
    console.log(`hexajobs: downloading ${url} ...`);
    await download(url, dest);
    chmodSync(dest, 0o755);
    const check = spawnSync(dest, ['--version'], { encoding: 'utf8' });
    if (check.status !== 0) throw new Error('--version check failed after download');
    console.log(`hexajobs: installed ${(check.stdout || '').trim()}`);
  } catch (err) {
    const built = tryBuildFromSource(pkgDir, arch);
    if (built && existsSync(built)) {
      copyFileSync(built, dest);
      chmodSync(dest, 0o755);
      console.log('hexajobs: built from source and installed.');
      return;
    }
    console.warn(
      `hexajobs: binary not installed (${err.message}).\n` +
        'To fix: publish/find a GitHub Release for ' +
        `v${pkg.version}, or install Go 1.24+ and run 'make build-linux-${arch}' in the package directory.`
    );
  }
}

main();
