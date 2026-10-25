import { spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { apiDirectory, binaryPath, buildAPI } from './api.mjs';

buildAPI();
const root = fileURLToPath(new URL('../', import.meta.url));
const webDirectory = fileURLToPath(new URL('../apps/web/', import.meta.url));
const next = fileURLToPath(new URL('../apps/web/node_modules/next/dist/bin/next', import.meta.url));
const children = [
  spawn(binaryPath, [], {
    cwd: apiDirectory,
    stdio: 'inherit',
    env: { ...process.env, APP_ENV: 'development', PORT: '8080' },
  }),
  spawn(process.execPath, [next, 'dev', '--port', '3000'], {
    cwd: webDirectory,
    stdio: 'inherit',
    env: process.env,
  }),
];
let stopping = false;
function stop(code = 0) {
  if (stopping) return;
  stopping = true;
  for (const child of children) {
    if (!child.pid) continue;
    if (process.platform === 'win32') {
      spawn('taskkill', ['/PID', String(child.pid), '/T', '/F'], { cwd: root, stdio: 'ignore' });
    } else child.kill('SIGTERM');
  }
  setTimeout(() => process.exit(code), 700);
}
for (const child of children) {
  child.on('error', (error) => {
    console.error(error.message);
    stop(1);
  });
  child.on('exit', (code) => stop(code || 0));
}
process.on('SIGINT', () => stop());
process.on('SIGTERM', () => stop());
