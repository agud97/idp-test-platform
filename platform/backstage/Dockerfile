FROM ghcr.io/backstage/backstage:latest

WORKDIR /app/packages/backend

RUN yarn add @backstage/plugin-auth-backend-module-oidc-provider@^0.4.2 sql.js@^1.13.0 marked@^12.0.2

WORKDIR /app

COPY docs/runbooks /app/docs/runbooks
COPY catalog /app/catalog
COPY templates /app/templates
COPY platform/backstage/custom-actions/gitops-repo-push-module.cjs /app/custom/gitops-repo-push-module.cjs
COPY platform/backstage/custom-actions/migration-dashboard-module.cjs /app/custom/migration-dashboard-module.cjs
COPY platform/backstage/custom-actions/techdocs-runbooks-module.cjs /app/custom/techdocs-runbooks-module.cjs

RUN node - <<'NODE'
const fs = require('fs');
const path = '/app/packages/backend/dist/index.cjs.js';
const source = fs.readFileSync(path, 'utf8');
const needle = "backend.add(import('@backstage/plugin-auth-backend-module-guest-provider'));";
const addition = "backend.add(import('@backstage/plugin-auth-backend-module-oidc-provider'));\nbackend.add(import('/app/custom/gitops-repo-push-module.cjs'));\nbackend.add(import('/app/custom/migration-dashboard-module.cjs'));\nbackend.add(import('/app/custom/techdocs-runbooks-module.cjs'));";
if (!source.includes(needle)) {
  throw new Error('guest provider hook not found');
}
if (source.includes(addition)) {
  process.exit(0);
}
fs.writeFileSync(path, source.replace(needle, `${needle}\n${addition}`));
NODE

# Patch frontend: replace hardcoded guest sign-in page with oidc
RUN node - <<'NODE'
const fs = require('fs');
const dir = '/app/packages/app/dist/static';
const files = fs.readdirSync(dir).filter(f => f.startsWith('main.') && f.endsWith('.js') && !f.endsWith('.map'));
if (files.length === 0) throw new Error('main.js not found in ' + dir);
const file = dir + '/' + files[0];
const src = fs.readFileSync(file, 'utf8');
const from = 'auto:!0,providers:["guest"]';
const to   = 'auto:!0,providers:["oidc"]';
if (!src.includes(from)) { console.log('already patched or pattern not found, skipping'); process.exit(0); }
fs.writeFileSync(file, src.replace(from, to));
console.log('patched', file);
NODE
