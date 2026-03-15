# OIDC Login Debug Report

**Дата:** 2026-03-15
**Цель:** Настроить вход в Backstage через OIDC (Authentik) из браузера пользователя

---

## Итоговое состояние на момент отчёта

- Backstage доступен по `http://backstage.idp.local:7007` (LoadBalancer `89.108.100.41:7007`)
- Authentik доступен по `http://authentik-server.authentik.svc.cluster.local` (LoadBalancer `89.108.100.218:80`)
- OIDC backend работает: `/api/auth/oidc/start?env=production` → 302 → Authentik
- Пользователь успешно логинится на Authentik (`akadmin` / `Admin1234!`)
- **Проблема не решена:** после успешного логина пользователь не попадает внутрь Backstage

---

## Шаг 1 — Экспозиция сервисов

### Проблема

Backstage и Authentik изначально были задеплоены без публичного доступа. Backstage имел тип сервиса по умолчанию (ClusterIP). Authentik — аналогично.

### Что делал

Изначально выбрал **NodePort** (ошибочно), полагая что OpenStack-ноды с внутренними IP `192.168.2.x` не поддерживают LoadBalancer.

```bash
kubectl get nodes -o jsonpath='{.items[*].spec.providerID}'
# → openstack:///xxxxxxxx-xxxx-...
```

Обнаружил `providerID: openstack:///...` — значит Cloud Controller Manager присутствует и LoadBalancer поддерживается.

### Исправление

Переключил оба сервиса на `type: LoadBalancer` в Helm values:

```yaml
# platform/backstage/helm-values.yaml
service:
  type: LoadBalancer
  ports:
    backend: 7007

# platform/authentik/helm-values.yaml
server:
  service:
    type: LoadBalancer
```

```bash
git add platform/backstage/helm-values.yaml platform/authentik/helm-values.yaml
git commit -m "Switch Backstage and Authentik services to LoadBalancer"
git push origin feature/phase-1-foundation
```

Получили LoadBalancer IP:
```bash
kubectl get svc -n backstage
# → 89.108.100.41:7007

kubectl get svc -n authentik
# → 89.108.100.218:80
```

Проверка:
```bash
curl -o /dev/null -w "%{http_code}" http://89.108.100.41:7007
# → 200

curl -o /dev/null -w "%{http_code}" http://89.108.100.218:80
# → 302
```

---

## Шаг 2 — HTTPS upgrade в браузере

### Проблема

Браузер редиректил все запросы на `https://backstage.idp.local:7007/static/...` вместо HTTP. В консоли браузера:

```
Failed to load resource: https://backstage.idp.local:7007/static/runtime.230b72e0.js
```

### Причина

В Backstage backend включён CSP-заголовок `upgrade-insecure-requests`, который заставляет браузер апгрейдить все HTTP ресурсы до HTTPS.

### Исправление

```yaml
# platform/backstage/helm-values.yaml → appConfig
backend:
  csp:
    upgrade-insecure-requests: false
  headers:
    Strict-Transport-Security: false
  helmet:
    hsts: false
```

```bash
git add platform/backstage/helm-values.yaml
git commit -m "Disable CSP upgrade-insecure-requests and HSTS"
git push origin feature/phase-1-foundation
```

**Примечание:** `Strict-Transport-Security` и `helmet.hsts: false` — не сработали (захардкожены в prebuilt image). Но HSTS по HTTP браузерами игнорируется (RFC 6797), поэтому практически не мешает.

---

## Шаг 3 — "Enter as a Guest User" вместо OIDC

### Проблема

После открытия `http://backstage.idp.local:7007` в браузере отображалась только кнопка:
> _"Enter as a Guest User. You will not have a verified identity, meaning some features might be unavailable."_

OIDC sign-in не появлялся.

### Диагностика

Проверили ConfigMap:
```bash
kubectl get configmap -n backstage backstage-app-config -o jsonpath='{.data.app-config\.yaml}' | grep -A10 'auth:'
```

ConfigMap содержал корректный `signInPage: oidc` и `auth.providers.oidc`, но:

1. В base image `/app/app-config.yaml` есть `auth.providers.guest: {}` — он мёржится поверх
2. `signInPage` находился на корневом уровне, а не под `app:`

### Исправление 1 — отключить guest провайдер

```yaml
# platform/backstage/app-config.auth.yaml
auth:
  providers:
    guest: null   # ← отключить guest из base image
    oidc: ...
```

### Исправление 2 — перенести signInPage под app:

```yaml
app:
  signInPage: oidc   # ← было на корневом уровне, не читалось фронтендом
```

```bash
git add platform/backstage/app-config.auth.yaml
git commit -m "Disable guest provider and move signInPage under app key"
git push origin feature/phase-1-foundation

kubectl annotate application backstage -n argocd argocd.argoproj.io/refresh=hard
kubectl rollout restart deployment/backstage -n backstage
kubectl rollout status deployment/backstage -n backstage --timeout=90s
```

**Результат:** `guest: null` появился в ConfigMap, но в HTML фронтенда `signInPage` по-прежнему отсутствовал. Кнопка Guest осталась.

### Причина

Фронтенд prebuilt image скомпилирован с захардкоженным:
```js
// packages/app/dist/static/main.95f13c68.js
SignInPage:e=>(0,s.jsx)(eO.q4,{...e,auto:!0,providers:["guest"]})
```

Ни `app.signInPage`, ни `auth.providers.guest: null` не влияют на скомпилированный JS. Backstage 1.48.0 в этом образе не поддерживает конфигурацию sign-in page через app-config.

---

## Шаг 4 — Попытка патча скомпилированного main.js

### Идея

Заменить `providers:["guest"]` на `providers:["oidc"]` прямо в скомпилированном JS через патч в Dockerfile.

### Реализация

```dockerfile
# platform/backstage/Dockerfile
RUN node - <<'NODE'
const fs = require('fs');
const dir = '/app/packages/app/dist/static';
const files = fs.readdirSync(dir).filter(f => f.startsWith('main.') && f.endsWith('.js') && !f.endsWith('.map'));
const file = dir + '/' + files[0];
const src = fs.readFileSync(file, 'utf8');
fs.writeFileSync(file, src.replace('auto:!0,providers:["guest"]', 'auto:!0,providers:["oidc"]'));
NODE
```

```bash
docker build -f platform/backstage/Dockerfile \
  -t ghcr.io/agud97/idp-test-platform/backstage-oidc:phase-4-task-4-6-r2 .
docker push ghcr.io/agud97/idp-test-platform/backstage-oidc:phase-4-task-4-6-r2
# Обновили helm-values.yaml tag → r2, запушили
```

### Результат

**Белый экран.** Строка `"oidc"` не является валидным провайдером в карте `J`:

```js
// module-backstage.3be92c1c.js
J = {
  guest:  { Component: ..., loader: B },
  custom: { Component: ..., loader: V },
  common: { Component: ..., loader: T },
  // "oidc" — не существует
}
```

При попытке `J["oidc"].Component` → `TypeError`, React крашился → белый экран.

---

## Шаг 5 — Исследование внутренней структуры фронтенда

### Диагностика

Изучили скомпилированный `module-backstage.3be92c1c.js`:

```bash
kubectl exec -n backstage deployment/backstage -- node -e "
const fs = require('fs');
const src = fs.readFileSync('/app/packages/app/dist/static/module-backstage.3be92c1c.js', 'utf8');
// Найти определение функции ee (экспортируется как q4)
const idx = src.indexOf('function ee(');
console.log(src.slice(idx, idx+200));
"
# → function ee(e){return "provider" in e?(0,i.jsx)(X,{...e}):(0,i.jsx)(Z,{...e})}
```

**Архитектура:**
- `q4` → `ee(props)`: если `provider` (ед.ч.) в props → рендерит `X` (single provider), иначе → `Z` (multi-provider)
- `Z` принимает `providers: string[]`, ищет каждую строку в `J`
- `X` принимает `provider: {apiRef, title, message}`, вызывает `useApi(provider.apiRef)`

```bash
# Найти B loader (авто-вход)
kubectl exec -n backstage deployment/backstage -- node -e "
const fs = require('fs');
const src = fs.readFileSync('/app/packages/app/dist/static/module-backstage.3be92c1c.js', 'utf8');
const idx = src.indexOf(',B=async');
console.log(src.slice(idx, idx+400));
"
# → B=async e=>{ ... new P({provider:'guest',...}).getBackstageIdentity() ... }
```

**Вывод:** OIDC ApiRef не зарегистрирован в скомпилированном фронтенде. `@backstage/core-app-api` OAuth2 класс и `oidcAuthApiRef` отсутствуют в бандле. Невозможно использовать стандартный Backstage OIDC flow через `X` или `Z` без пересборки из исходников.

---

## Шаг 6 — Патч guest Component с popup + postMessage

### Идея

Заменить логику `guest` Component в `module-backstage.3be92c1c.js`:
- Открывать popup на `/api/auth/oidc/start?env=production`
- Слушать `window.postMessage` от popup (Backstage backend отправляет его через `sendWebMessageResponse`)
- Вызывать `onSignInSuccess` с данными из сообщения

Изучили `sendWebMessageResponse`:
```bash
kubectl exec -n backstage deployment/backstage -- \
  cat /app/node_modules/@backstage/plugin-auth-node/dist/flow/sendWebMessageResponse.cjs.js
```

Нашли точный формат:
```js
// popup отправляет два сообщения:
(window.opener).postMessage({'type': 'config_info', 'targetOrigin': origin}, '*');
(window.opener).postMessage(JSON.parse(authResponse), origin);
// authResponse содержит: { backstageIdentity, profile, providerInfo }
```

### Реализация (r3 → r4)

```dockerfile
# platform/backstage/Dockerfile — патч module-backstage.js
# 1. Заменить B loader: сначала пробовать OIDC refresh
const bTo = ',B=async e=>{try{const res=await fetch("/api/auth/oidc/refresh?env=production",{headers:{"X-Requested-With":"XMLHttpRequest"},credentials:"include"});if(res.ok){const d=await res.json();if(d&&d.backstageIdentity)return{getBackstageIdentity:async()=>d.backstageIdentity,getProfile:async()=>d.profile,signOut:async()=>{}};}}catch(err){}};';

# 2. Заменить guest Component: popup + postMessage listener
const gTo = 'guest:{Component:({onSignInStarted:e,onSignInSuccess:t,onSignInFailure:r})=>{
  const u=()=>{
    e();
    const p=window.open("/api/auth/oidc/start?env=production","_oidcLogin","width=600,height=700");
    if(!p){r();return;}
    let done=false;
    const cleanup=()=>{done=true;window.removeEventListener("message",msgHandler);clearInterval(tm);};
    const msgHandler=(ev)=>{
      if(done)return;
      const d=ev.data;
      if(d&&d.type==="config_info")return;  // игнорируем первое сообщение
      if(d&&(d.backstageIdentity||d.profile)){
        cleanup();
        t({getBackstageIdentity:async()=>d.backstageIdentity,getProfile:async()=>d.profile||{},signOut:async()=>{}});
      }else if(d&&d.error){cleanup();r();}
    };
    window.addEventListener("message",msgHandler);
    const tm=setInterval(async()=>{
      if(p.closed&&!done){  // fallback: popup закрыт без сообщения
        cleanup();
        // попытка через refresh endpoint
        ...
        r();
      }
    },500);
  };
  ...
},loader:B}';
```

```bash
docker build -f platform/backstage/Dockerfile \
  -t ghcr.io/agud97/idp-test-platform/backstage-oidc:phase-4-task-4-6-r4 .
docker push ghcr.io/agud97/idp-test-platform/backstage-oidc:phase-4-task-4-6-r4
# tag → r4, commit, push
kubectl annotate application backstage -n argocd argocd.argoproj.io/refresh=hard
```

### Результат

Пользователь успешно проходит аутентификацию на Authentik (логин `akadmin` / `Admin1234!`), popup закрывается, но **внутрь Backstage не попадает**.

---

## Текущая диагностика (на момент отчёта)

Из логов Backstage после успешного OIDC логина:

```
GET /api/auth/guest/refresh → 404   (старый код ещё в браузере?)
GET /api/catalog/entities?filter=...user:default/guest → 401
POST /api/permission/authorize → 401
```

**Наблюдения:**

1. `/api/auth/guest/refresh` вызывается — это либо старый кэшированный JS в браузере, либо `r4` ещё не применён
2. Фронтенд пытается работать с identity `user:default/guest` — что означает либо:
   - `onSignInSuccess` вызван с невалидным identity (нет JWT token)
   - Resolver `emailMatchingUserEntityProfileEmail` упал: пользователь `akadmin` (email `root@example.com`) не найден в Backstage catalog
3. Все API-запросы возвращают 401 — токен не проходит валидацию

---

## Корневые причины (гипотезы)

### Гипотеза A — Resolver failure

`signIn.resolvers[0].resolver: emailMatchingUserEntityProfileEmail` — ищет User entity в catalog по email. У `akadmin` email `root@example.com`. В catalog нет соответствующей User entity → resolver возвращает ошибку → `backstageIdentity` в postMessage содержит error вместо данных.

**Проверка:**
```bash
kubectl logs -n backstage deployment/backstage --tail=50 | grep -i 'resolver\|sign.in\|email'
```

### Гипотеза B — Кэш браузера

Пользователь получает старый `main.js` / `module-backstage.js` из кэша. Файлы имеют content-hash в именах, но содержимое изменилось без смены хэша → браузер отдаёт старый файл.

**Проверка:** открыть в инкогнито + DevTools → Network → Disable cache.

### Гипотеза C — postMessage origin mismatch

`sendWebMessageResponse` отправляет второе сообщение с target origin `appOrigin`. Если `appOrigin` не совпадает с `window.location.origin` (например, `http://localhost:7007` vs `http://backstage.idp.local:7007`), браузер блокирует доставку.

**Проверка:**
```bash
# Что установлено как appOrigin в OIDC handler:
kubectl logs -n backstage deployment/backstage | grep 'origin'
```

---

## Что нужно сделать дальше

1. **Проверить resolver:** добавить fallback resolver `allowedUsernamesByEnvironment` или `emailLocalPartMatchingUserEntityName` в `app-config.auth.yaml`
2. **Проверить origin:** убедиться что `app.baseUrl` и Authentik redirect URI используют `http://backstage.idp.local:7007`
3. **Добавить User entity** для `akadmin` в catalog или сменить resolver на `usernameMatchingUserEntityAnnotation`
4. **Проверить кэш:** открыть в инкогнито с `Disable Cache` в DevTools

---

## Образы

| Tag | Что изменено |
|-----|-------------|
| `phase-4-task-4-6-r1` | Исходный образ |
| `phase-4-task-4-6-r2` | Патч `providers:["oidc"]` → белый экран |
| `phase-4-task-4-6-r3` | Патч guest Component с popup (без postMessage) |
| `phase-4-task-4-6-r4` | Патч guest Component с popup + postMessage listener |

## Конфигурация /etc/hosts (браузер пользователя)

```
89.108.100.41   backstage.idp.local
89.108.100.218  authentik-server.authentik.svc.cluster.local
```
