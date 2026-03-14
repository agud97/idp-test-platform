FROM ghcr.io/backstage/backstage:latest

WORKDIR /app/packages/backend

RUN yarn add @backstage/plugin-auth-backend-module-oidc-provider@^0.4.2

RUN node - <<'NODE'
const fs = require('fs');
const path = '/app/packages/backend/dist/index.cjs.js';
const source = fs.readFileSync(path, 'utf8');
const needle = "backend.add(import('@backstage/plugin-auth-backend-module-guest-provider'));";
const addition = "backend.add(import('@backstage/plugin-auth-backend-module-oidc-provider'));";
if (!source.includes(needle)) {
  throw new Error('guest provider hook not found');
}
if (source.includes(addition)) {
  process.exit(0);
}
fs.writeFileSync(path, source.replace(needle, `${needle}\n${addition}`));
NODE

WORKDIR /app
