#!/usr/bin/env node
'use strict';

const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const https = require('https');
const http = require('http');
const { URL } = require('url');

const VERSION = require('./package.json').version;
const REPO = 'Mohsinsiddi/w3cli';

// Map Node.js platform/arch to Go binary naming convention.
const PLATFORM_MAP = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'windows',
};

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64',
  arm: 'arm',
};

function getBinaryName() {
  const plat = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];

  if (!plat || !arch) {
    throw new Error(
      `Unsupported platform: ${process.platform}/${process.arch}. ` +
      'Please build from source: https://github.com/' + REPO
    );
  }

  const ext = process.platform === 'win32' ? '.exe' : '';
  return `w3cli-${plat}-${arch}${ext}`;
}

function getBinaryDestPath() {
  const ext = process.platform === 'win32' ? '.exe' : '';
  return path.join(__dirname, 'bin', `w3cli${ext}`);
}

function downloadFile(url, dest, redirectCount = 0) {
  return new Promise((resolve, reject) => {
    if (redirectCount > 5) {
      return reject(new Error('Too many redirects'));
    }

    const parsedURL = new URL(url);
    const lib = parsedURL.protocol === 'https:' ? https : http;

    const req = lib.get(url, (res) => {
      // Follow redirects (GitHub releases use redirects).
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        return resolve(downloadFile(res.headers.location, dest, redirectCount + 1));
      }

      if (res.statusCode !== 200) {
        return reject(new Error(`Download failed: HTTP ${res.statusCode} from ${url}`));
      }

      const file = fs.createWriteStream(dest);
      res.pipe(file);
      file.on('finish', () => {
        file.close(() => resolve());
      });
      file.on('error', (err) => {
        fs.unlink(dest, () => {}); // clean up partial file
        reject(err);
      });
    });

    req.on('error', reject);
    req.setTimeout(60000, () => {
      req.destroy();
      reject(new Error('Download timed out after 60s'));
    });
  });
}

async function install() {
  const binaryName = getBinaryName();
  const dest = getBinaryDestPath();
  const downloadURL =
    `https://github.com/${REPO}/releases/download/v${VERSION}/${binaryName}`;

  // Ensure bin/ directory exists.
  const binDir = path.dirname(dest);
  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  // Skip download if binary already exists and is executable.
  if (fs.existsSync(dest)) {
    console.log(`w3cli already installed at ${dest}`);
    return;
  }

  console.log(`Downloading w3cli v${VERSION} for ${process.platform}/${process.arch}...`);
  console.log(`  Source: ${downloadURL}`);

  try {
    await downloadFile(downloadURL, dest);

    // Make the binary executable on Unix systems.
    if (process.platform !== 'win32') {
      fs.chmodSync(dest, 0o755);
    }

    console.log(`✓ w3cli installed successfully to ${dest}`);
  } catch (err) {
    // Don't hard-fail the npm install — warn the user instead.
    console.error(`\nWarning: Failed to download w3cli binary: ${err.message}`);
    console.error('You can manually download the binary from:');
    console.error(`  https://github.com/${REPO}/releases/tag/v${VERSION}`);
    console.error(`Place it at: ${dest}`);
    console.error('Or build from source:');
    console.error(`  git clone https://github.com/${REPO}.git && cd w3cli && go build -o w3cli .`);
  }
}

install();
