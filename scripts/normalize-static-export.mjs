import { copyFileSync, existsSync, readFileSync, readdirSync } from 'node:fs';
import { dirname, join, relative, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

// Next 16 static exports built on Windows can leave nested RSC segment files
// where the browser expects dot-separated filenames. Add the expected aliases
// without modifying Next itself or deleting its output. Correct exports are a
// no-op. Remove once the pinned Next version fixes this upstream issue:
// https://github.com/vercel/next.js/issues/85374
const output = fileURLToPath(new URL('../apps/web/out/', import.meta.url));
let copied = 0;

function addAliases(segmentRoot, directory = segmentRoot) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const source = join(directory, entry.name);
    if (entry.isDirectory()) {
      addAliases(segmentRoot, source);
    } else if (entry.isFile() && entry.name.endsWith('.txt')) {
      const parent = dirname(segmentRoot);
      const filename = relative(parent, source).split(sep).join('.');
      const destination = join(parent, filename);
      if (existsSync(destination)) {
        if (!readFileSync(destination).equals(readFileSync(source))) {
          throw new Error(`Conflicting static segment: ${filename}`);
        }
      } else {
        copyFileSync(source, destination);
        copied++;
      }
    }
  }
}

function visit(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const child = join(directory, entry.name);
    if (entry.name.startsWith('__next.')) addAliases(child);
    else visit(child);
  }
}

visit(output);
console.log(`Static export: added ${copied} Windows RSC filename aliases.`);
