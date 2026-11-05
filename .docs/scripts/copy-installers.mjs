import {copyFileSync, existsSync} from 'node:fs';
import {dirname, join} from 'node:path';
import {fileURLToPath} from 'node:url';

const docsRoot = join(dirname(fileURLToPath(import.meta.url)), '..');
const repositoryRoot = join(docsRoot, '..');

for (const installer of ['install.sh', 'install.ps1']) {
  const source = join(repositoryRoot, installer);
  if (existsSync(source)) {
    copyFileSync(source, join(docsRoot, 'static', installer));
  }
}
