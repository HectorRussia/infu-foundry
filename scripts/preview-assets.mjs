import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { resolve, join } from 'node:path';
import { parse } from 'jsonc-parser';

// Exercise real Worker/asset routing without Docker. API intentionally returns
// 503 in this mode; this is NOT a substitute for the full container preview.
const directory = fileURLToPath(new URL('../apps/gateway/', import.meta.url));
const errors = [];
const config = parse(readFileSync(join(directory, 'wrangler.jsonc'), 'utf8'), errors, {
  allowTrailingComma: true,
});
if (errors.length) throw new Error('Invalid Wrangler configuration');
config.main = resolve(directory, config.main);
config.assets.directory = resolve(directory, config.assets.directory);
for (const container of config.containers) container.image = resolve(directory, container.image);
config.dev = { enable_containers: false };
delete config.$schema;
const generated = join(directory, '.wrangler', 'assets-preview.json');
mkdirSync(join(directory, '.wrangler'), { recursive: true });
writeFileSync(generated, JSON.stringify(config, null, 2));
const cli = join(directory, 'node_modules', 'wrangler', 'bin', 'wrangler.js');
const child = spawn(
  process.execPath,
  [cli, 'dev', '--local', '--config', generated, '--port', '8787'],
  { cwd: directory, stdio: 'inherit' },
);
child.on('error', (error) => {
  console.error(error.message);
  process.exitCode = 1;
});
child.on('exit', (code) => {
  process.exitCode = code || 0;
});
for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, () => child.kill(signal));
