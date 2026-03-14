FROM ghcr.io/backstage/backstage:latest

WORKDIR /app/packages/backend

RUN yarn add @backstage/plugin-auth-backend-module-oidc-provider@^0.4.2 sql.js@^1.13.0

WORKDIR /app

COPY catalog /app/catalog
COPY templates /app/templates
COPY platform/backstage/custom-actions/gitops-repo-push-module.cjs /app/custom/gitops-repo-push-module.cjs
COPY platform/backstage/custom-actions/migration-dashboard-module.cjs /app/custom/migration-dashboard-module.cjs

RUN node - <<'NODE'
const fs = require('fs');
const path = '/app/packages/backend/dist/index.cjs.js';
const source = fs.readFileSync(path, 'utf8');
const needle = "backend.add(import('@backstage/plugin-auth-backend-module-guest-provider'));";
const addition = "backend.add(import('@backstage/plugin-auth-backend-module-oidc-provider'));\nbackend.add(import('/app/custom/gitops-repo-push-module.cjs'));\nbackend.add(import('/app/custom/migration-dashboard-module.cjs'));";
if (!source.includes(needle)) {
  throw new Error('guest provider hook not found');
}
if (source.includes(addition)) {
  process.exit(0);
}
fs.writeFileSync(path, source.replace(needle, `${needle}\n${addition}`));
NODE
