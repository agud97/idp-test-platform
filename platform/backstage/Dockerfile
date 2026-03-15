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

# Patch frontend: replace guest sign-in component with OIDC popup flow
RUN node - <<'NODE'
const fs = require('fs');
const dir = '/app/packages/app/dist/static';

// Patch module-backstage: replace guest Component + B loader with OIDC
const mbFiles = fs.readdirSync(dir).filter(f => f.startsWith('module-backstage.') && f.endsWith('.js') && !f.endsWith('.map'));
if (mbFiles.length === 0) throw new Error('module-backstage.js not found');
const mbFile = dir + '/' + mbFiles[0];
let mb = fs.readFileSync(mbFile, 'utf8');

// 1. Replace B loader (auto-sign-in) to try OIDC session first
const bFrom = ',B=async e=>{let t="true"===localStorage.getItem("enableLegacyGuestToken"),r=new P({provider:"guest",discoveryApi:e.get(O.I)}),i=await L(r).catch(e=>{console.warn(`Failed to sign in as a guest, ${e}`)});if(i||t){if(i&&t){if(!confirm("You are currently using the legacy guest token, but you have the new guest backend module installed. Do you want to use the new module?"))return new o;localStorage.removeItem("enableLegacyGuestToken")}else if(t)return new o;return r}};';
const bTo   = ',B=async e=>{try{const res=await fetch("/api/auth/oidc/refresh?env=production",{headers:{"X-Requested-With":"XMLHttpRequest"},credentials:"include"});if(res.ok){const d=await res.json();if(d&&d.backstageIdentity)return{getBackstageIdentity:async()=>d.backstageIdentity,getProfile:async()=>d.profile,signOut:async()=>{}};}}catch(err){}};';
if (!mb.includes(bFrom)) throw new Error('B loader pattern not found');
mb = mb.replace(bFrom, bTo);

// 2. Replace guest Component to open OIDC popup
const gFrom = 'guest:{Component:({onSignInStarted:e,onSignInSuccess:t,onSignInFailure:r})=>{let a=(0,n.gf)(O.I),[s,l]=(0,N.A)("enableLegacyGuestToken"),{t:c}=(0,E.i)(C.O),u=async()=>{e();let i=new P({provider:"guest",discoveryApi:a});if(!await L(i).catch(e=>{console.warn(`Failed to sign in as a guest, ${e}`)})){if(confirm("Failed to sign in as a guest using the auth backend. Do you want to fallback to the legacy guest token?")){l(!0),t(new o);return}throw r(),Error("You cannot sign in as a guest, you must either enable the legacy guest token or configure the auth backend to support guest sign in.")}t(i)};return(0,i.jsx)(k,{children:(0,i.jsx)(x.n,{title:"Guest",variant:"fullHeight",actions:(0,i.jsx)(d.A,{color:"primary",variant:"outlined",onClick:u,children:c("signIn.guestProvider.enter")}),children:(0,i.jsx)(p.A,{variant:"body1",style:{whiteSpace:"pre-line"},children:c("signIn.guestProvider.subtitle")})})})},loader:B}';
const gTo   = 'guest:{Component:({onSignInStarted:e,onSignInSuccess:t,onSignInFailure:r})=>{const u=()=>{e();const p=window.open("/api/auth/oidc/start?env=production","_oidcLogin","width=600,height=700");if(!p){r();return;}const tm=setInterval(async()=>{if(p.closed){clearInterval(tm);try{const res=await fetch("/api/auth/oidc/refresh?env=production",{headers:{"X-Requested-With":"XMLHttpRequest"},credentials:"include"});if(!res.ok){r();return;}const d=await res.json();if(!d||!d.backstageIdentity){r();return;}t({getBackstageIdentity:async()=>d.backstageIdentity,getProfile:async()=>d.profile,signOut:async()=>{}});}catch(err){r();}}},500);};if(!window.__oidcSignInStarted){window.__oidcSignInStarted=true;setTimeout(u,100);}return h.createElement("div",{style:{display:"flex",flexDirection:"column",alignItems:"center",justifyContent:"center",height:"100vh"}},h.createElement("button",{onClick:()=>{window.__oidcSignInStarted=false;u();},style:{padding:"12px 24px",fontSize:"16px",cursor:"pointer"}},"Sign in with OIDC"));},loader:B}';
if (!mb.includes(gFrom)) throw new Error('guest Component pattern not found');
mb = mb.replace(gFrom, gTo);

fs.writeFileSync(mbFile, mb);
console.log('patched', mbFile);
NODE
