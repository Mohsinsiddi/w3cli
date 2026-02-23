#!/usr/bin/env node
'use strict';

const { spawnSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const ext = process.platform === 'win32' ? '.exe' : '';
const binary = path.join(__dirname, `w3cli${ext}`);

if (!fs.existsSync(binary)) {
  console.error(
    'w3cli binary not found. Try reinstalling:\n' +
    '  npm install -g w3cli\n\n' +
    'Or build from source:\n' +
    '  git clone https://github.com/Mohsinsiddi/w3cli.git\n' +
    '  cd w3cli && go build -o w3cli .\n'
  );
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), {
  stdio: 'inherit',
  env: process.env,
});

if (result.error) {
  console.error(`Failed to spawn w3cli: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status ?? 0);
