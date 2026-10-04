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

const AUTH_HOSTS = new Set(['github.com', 'api.github.com']);

function authHeaders(url) {
  try {
    if (process.env.GITHUB_TOKEN && AUTH_HOSTS.has(new URL(url).hostname)) {
      return { Authorization: `Bearer ${process.env.GITHUB_TOKEN}` };
    }
  } catch {
    // fall through to no-auth request below
  }
  return {};
}

function getJson(url) {
  return new Promise((resolve, reject) => {
    get(
      url,
      { headers: { Accept: 'application/vnd.github+json', 'User-Agent': 'hexajobs-cli-installer', ...authHeaders(url) } },
      (res) => {
        let body = '';
        res.on('data', (c) => (body += c));
        res.on('end', () => {
          if (res.statusCode !== 200) {
            reject(new Error(`HTTP ${res.statusCode} for ${url}`));
            return;
          }
          try {
            resolve(JSON.parse(body));
          } catch {
            reject(new Error(`invalid JSON from ${url}`));
          }
        });
      }
    ).on('error', reject);
  });
}

async function releaseAssetId(tag, name) {
  const rel = await getJson(`https://api.github.com/repos/${REPO}/releases/tags/${tag}`);
  const asset = (rel.assets || []).find((a) => a.name === name);
  if (!asset) throw new Error(`asset ${name} not found in release ${tag}`);
  return asset.id;
}

function download(url, dest, redirects = 5, accept) {
  return new Promise((resolve, reject) => {
    const headers = { 'User-Agent': 'hexajobs-cli-installer', ...authHeaders(url) };
    if (accept) headers.Accept = accept;
    get(url, { headers }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        if (redirects === 0) {
          reject(new Error('too many redirects'));
          return;
        }
        res.resume();
        // Auth headers are recomputed per hop, so the token is never
        // forwarded to the signed objects.githubusercontent.com URL.
        download(res.headers.location, dest, redirects - 1, accept).then(resolve, reject);
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

async function downloadReleaseBinary(tag, name, dest) {
  // Try the public URL first (works for public repos, no token needed).
  // With GITHUB_TOKEN set, fall back to the API so private repos and
  // API rate-limit cases keep working; auth headers are recomputed per
  // redirect hop and never forwarded to signed download URLs.
  if (process.env.GITHUB_TOKEN) {
    const id = await releaseAssetId(tag, name);
    await download(
      `https://api.github.com/repos/${REPO}/releases/assets/${id}`,
      dest,
      5,
      'application/octet-stream'
    );
    return `GitHub Release ${tag} (authenticated API)`;
  }
  const url = `https://github.com/${REPO}/releases/download/${tag}/${name}`;
  console.log(`hexajobs: downloading ${url} ...`);
  await download(url, dest);
  return `GitHub Release ${tag}`;
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

  const tag = `v${pkg.version}`;
  const name = `hexajobs-linux-${arch}`;
  try {
    const source = await downloadReleaseBinary(tag, name, dest);
    chmodSync(dest, 0o755);
    const check = spawnSync(dest, ['--version'], { encoding: 'utf8' });
    if (check.status !== 0) throw new Error('--version check failed after download');
    console.log(`hexajobs: installed ${(check.stdout || '').trim()} from ${source}`);
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
        `To fix: ensure GitHub Release ${tag} has asset ${name}` +
        ' (set GITHUB_TOKEN with repo access if the repo is private),' +
        ` or install Go 1.24+ and run 'make build-linux-${arch}' in the package directory.`
    );
  }
}

main();
