import { existsSync, mkdirSync } from 'node:fs';
import { spawnSync, spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';

export const apiDirectory = fileURLToPath(new URL('../apps/api/', import.meta.url));

// Local-only secrets (gitignored); variables already set in the shell take precedence.
const envFile = fileURLToPath(new URL('../apps/api/.env', import.meta.url));
if (existsSync(envFile)) process.loadEnvFile(envFile);
export const binaryPath = fileURLToPath(
  new URL(`../apps/api/bin/server${process.platform === 'win32' ? '.exe' : ''}`, import.meta.url),
);

export function buildAPI() {
  mkdirSync(new URL('../apps/api/bin/', import.meta.url), { recursive: true });
  const result = spawnSync('go', ['build', '-o', binaryPath, './cmd/server'], {
    cwd: apiDirectory,
    stdio: 'inherit',
  });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error('Go build failed');
}

const command = process.argv[2];
if (command === 'check') {
  for (const args of [
    ['vet', './...'],
    ['test', './...'],
  ]) {
    const result = spawnSync('go', args, { cwd: apiDirectory, stdio: 'inherit' });
    if (result.error || result.status !== 0) process.exit(result.status || 1);
  }
} else if (command === 'build') {
  buildAPI();
  const result = spawnSync(
    'go',
    ['build', '-o', `bin/migrate${process.platform === 'win32' ? '.exe' : ''}`, './cmd/migrate'],
    { cwd: apiDirectory, stdio: 'inherit' },
  );
  if (result.error || result.status !== 0) process.exit(result.status || 1);
} else if (command === 'migrate') {
  const result = spawnSync('go', ['run', './cmd/migrate', ...process.argv.slice(3)], {
    cwd: apiDirectory,
    stdio: 'inherit',
    env: process.env,
  });
  if (result.error) console.error(result.error.message);
  process.exitCode = result.status ?? 1;
} else if (command === 'create-user') {
  const result = spawnSync('go', ['run', './cmd/createuser', ...process.argv.slice(3)], {
    cwd: apiDirectory,
    stdio: 'inherit',
    env: process.env,
  });
  if (result.error) console.error(result.error.message);
  process.exitCode = result.status ?? 1;
} else if (command === 'run') {
  buildAPI();
  const server = spawn(binaryPath, [], { cwd: apiDirectory, stdio: 'inherit', env: process.env });
  for (const signal of ['SIGINT', 'SIGTERM']) process.on(signal, () => server.kill(signal));
  server.on('exit', (code) => process.exit(code || 0));
}
