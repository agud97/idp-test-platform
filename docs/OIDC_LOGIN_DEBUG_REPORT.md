# OIDC Login Debug Report

**Дата:** 2026-03-15
**Цель:** Настроить вход в Backstage через OIDC (Authentik) из браузера пользователя
**Итог:** **РЕШЕНО** — полный OIDC логин работает. Логин `akadmin` / `Admin1234!`.

---

## Итоговое состояние

- Backstage доступен по `http://backstage.idp.local:7007` (LoadBalancer `89.108.100.41:7007`)
- Authentik доступен по `http://89.108.100.218:80`
- OIDC логин полностью работает: кнопка "Sign in with OIDC" → popup Authentik → ввод логина/пароля → redirect обратно → Backstage открывается
- Учётные данные: `akadmin` / `Admin1234!`

---

## Шаг 1 — Экспозиция сервисов

### Проблема

Backstage и Authentik изначально задеплоены без публичного доступа (ClusterIP).

### Диагностика

```bash
kubectl get nodes -o jsonpath='{.items[*].spec.providerID}'
# → openstack:///xxxxxxxx-xxxx-...
```

`providerID: openstack://` → Cloud Controller Manager присутствует → LoadBalancer поддерживается.

### Исправление

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

Результат:
```
kubectl get svc -n backstage  → 89.108.100.41:7007
kubectl get svc -n authentik  → 89.108.100.218:80
```

---

## Шаг 2 — HTTPS upgrade-insecure-requests

### Проблема

Браузер редиректил все запросы на HTTPS:
```
Failed to load resource: https://backstage.idp.local:7007/static/runtime.230b72e0.js
```

### Причина

CSP-заголовок `upgrade-insecure-requests` в Backstage backend заставлял браузер апгрейдить все HTTP-ресурсы до HTTPS.

### Исправление

```yaml
# platform/backstage/helm-values.yaml → appConfig → backend
backend:
  csp:
    upgrade-insecure-requests: false
  headers:
    Strict-Transport-Security: false
  helmet:
    hsts: false
```

**Примечание:** `Strict-Transport-Security` и `helmet.hsts` захардкожены в prebuilt image и не меняются через app-config. Но HSTS по HTTP браузерами игнорируется (RFC 6797), поэтому на практике не мешает.

---

## Шаг 3 — Guest вместо OIDC на странице входа

### Проблема

После открытия `http://backstage.idp.local:7007` — только кнопка "Enter as a Guest User". OIDC не отображается.

### Причина

Фронтенд Backstage 1.48.0 (prebuilt image, rspack bundler) скомпилирован с захардкоженным:
```js
// packages/app/dist/static/main.*.js
SignInPage: e => (0,s.jsx)(eO.q4, {...e, auto:!0, providers:["guest"]})
```

Ни `app.signInPage: oidc`, ни `auth.providers.guest: null` не влияют на скомпилированный JS. Стандартная конфигурация sign-in page через app-config не работает в этом образе.

Дополнительно: в base image `app-config.yaml` есть `auth.providers.guest: {}` — он мёржится поверх. `signInPage` было указано на корневом уровне YAML вместо `app.signInPage`.

### Попытка 1 (r2) — patch providers:["oidc"]

```dockerfile
RUN node - <<'NODE'
const fs = require('fs');
const dir = '/app/packages/app/dist/static';
const files = fs.readdirSync(dir).filter(f => f.startsWith('main.') && f.endsWith('.js'));
const file = dir + '/' + files[0];
const src = fs.readFileSync(file, 'utf8');
fs.writeFileSync(file, src.replace('auto:!0,providers:["guest"]', 'auto:!0,providers:["oidc"]'));
NODE
```

**Результат: белый экран.** В `module-backstage.*.js` есть карта провайдеров `J`:
```js
J = {
  guest:  { Component: ..., loader: B },
  custom: { Component: ..., loader: V },
  common: { Component: ..., loader: T },
  // "oidc" — не существует
}
```
При обращении к `J["oidc"].Component` — TypeError → React crash → белый экран.

### Решение — патч guest Component с popup + postMessage

Изучена внутренняя структура `module-backstage.*.js`:

```bash
kubectl exec -n backstage deployment/backstage -- node -e "
const fs = require('fs');
const src = fs.readFileSync('/app/packages/app/dist/static/module-backstage.3be92c1c.js', 'utf8');
// Структура карты провайдеров J
const idx = src.indexOf(',B=async');
console.log(src.slice(idx, idx+400));
"
```

- `B` = loader: автоматически пробует войти (изначально — как guest)
- `guest.Component` = UI компонент кнопки входа

Изучен `sendWebMessageResponse`:
```bash
kubectl exec -n backstage deployment/backstage -- \
  cat /app/node_modules/@backstage/plugin-auth-node/dist/flow/sendWebMessageResponse.cjs.js
```

Popup отправляет два сообщения родительскому окну:
1. `{type: 'config_info', targetOrigin: origin}` — игнорировать
2. `{type: 'authorization_response', response: {backstageIdentity, profile, providerInfo}}` — данные идентичности

**Исправление:** заменить `guest.Component` на компонент, открывающий OIDC popup, а `B` loader — на проверку сессии через `/api/auth/oidc/refresh`.

---

## Шаг 4 — Кэш браузера (проблема r3/r4 → r8)

### Проблема

Даже после пуша нового Docker-образа с патчем браузер продолжал отдавать старый `module-backstage.3be92c1c.js`. Backstage отдаёт статические JS-файлы с:
```
Cache-Control: public, max-age=1209600   # 2 недели
```

Имя файла содержит content-hash (`3be92c1c`), но при патче в Dockerfile содержимое изменялось без смены хэша в имени → браузер считал файл неизменным и брал из кэша.

### Попытка r6 — query param `?v=r6` в index.html

Добавить `?v=r6` к URL скрипта в `index.html`:
```js
html = html.replace(mbBase, mbBase + '?v=r6');
```

**Не сработало:** Backstage app-backend не сервирует `index.html` напрямую. Он обрабатывает `index.html.tmpl` — шаблон, в который инжектируется конфиг — и генерирует ответ заново на каждый запрос. Изменения в `index.html` игнорируются.

### Попытка r7 — rename + patch index.html

Переименовать файл в `module-backstage.oidcpatch.js` и патчить `index.html`.

**Не сработало:** по той же причине — app-backend использует `index.html.tmpl`, а не `index.html`.

### Исправление r8 — rename + patch ОБОИХ файлов

```dockerfile
# В Dockerfile — патч module-backstage.js, затем:

# 3. Rename patched file to bust browser cache
const newMbFile = dir + '/module-backstage.oidcpatch.js';
fs.renameSync(mbFile, newMbFile);
const mapFile = mbFile + '.map';
if (fs.existsSync(mapFile)) fs.renameSync(mapFile, newMbFile + '.map');

# 4. Patch BOTH index.html AND index.html.tmpl
const mbBase = mbFiles[0]; // e.g. module-backstage.3be92c1c.js
const mbRe = new RegExp(mbBase.replace(/\./g, '\\.'), 'g');
for (const htmlFile of ['/app/packages/app/dist/index.html', '/app/packages/app/dist/index.html.tmpl']) {
  if (fs.existsSync(htmlFile)) {
    let html = fs.readFileSync(htmlFile, 'utf8');
    html = html.replace(mbRe, 'module-backstage.oidcpatch.js');
    fs.writeFileSync(htmlFile, html);
  }
}
```

**Сработало:** сервер начал отдавать `module-backstage.oidcpatch.js` в HTML — новое имя файла, кэш не попадает.

Проверка:
```bash
curl -s http://89.108.100.41:7007 | grep 'module-backstage'
# → module-backstage.oidcpatch.js

kubectl exec deployment/backstage -n backstage -- \
  grep -c 'oidc/refresh' /app/packages/app/dist/static/module-backstage.oidcpatch.js
# → 1 (патч применён)

kubectl exec deployment/backstage -n backstage -- \
  grep -c 'enableLegacyGuestToken' /app/packages/app/dist/static/module-backstage.oidcpatch.js
# → 0 (старый guest код удалён)
```

---

## Шаг 5 — Resolver emailMatchingUserEntityProfileEmail

### Проблема

После успешного OIDC логина на Authentik Backstage возвращал ошибку. Из логов:
```
Error: Failed to sign in as user: could not find user entity with email root@example.com
```

### Причина

`signIn.resolvers[0].resolver: emailMatchingUserEntityProfileEmail` ищет User entity в каталоге Backstage по email. Пользователь `akadmin` в Authentik имеет email `root@example.com`. В каталоге не было соответствующей User entity.

### Исправление

```yaml
# catalog/all-components.yaml — добавить User entity
apiVersion: backstage.io/v1alpha1
kind: User
metadata:
  name: akadmin
spec:
  profile:
    displayName: Admin
    email: root@example.com
  memberOf:
    - platform
```

```yaml
# platform/backstage/app-config.catalog.yaml — разрешить User kind
catalog:
  locations:
    - type: file
      target: /app/catalog/all-components.yaml
      rules:
        - allow:
            - Component
            - System
            - Group
            - User   # ← добавлено
```

---

## Шаг 6 — prompt=none → login_required

### Проблема

После всех предыдущих исправлений popup открывался и почти сразу закрывался (~300ms). Из логов:
```
2026-03-15T16:32:40.927Z [302] /api/auth/oidc/start?env=production
2026-03-15T16:32:41.310Z [200] /api/auth/oidc/handler/frame?error=login_required&...
```

### Причина

Backstage по умолчанию добавляет `prompt=none` к OAuth2 authorize-запросу, когда `prompt:` не задан в конфиге. Authentik при `prompt=none` немедленно возвращает `login_required`, если у пользователя нет активной сессии.

Проверка:
```bash
curl -sv "http://89.108.100.41:7007/api/auth/oidc/start?env=production" 2>&1 | grep Location
# → &prompt=none& в URL редиректа на Authentik
```

### Попытка — prompt: login

Добавление `prompt: login` в `app-config.auth.yaml` вызвало бесконечный цикл: после успешного логина Authentik с `prompt=login` redirect обратно с кодом авторизации → Backstage снова открывает `/authorize?prompt=login` → Authentik снова показывает форму логина → бесконечно.

Причина: Authentik всегда инициирует новый authentication flow при `prompt=login`, не проверяя наличие существующей сессии.

### Исправление — prompt: select_account

```yaml
# platform/backstage/app-config.auth.yaml
oidc:
  production:
    metadataUrl: ...
    clientId: backstage
    clientSecret: ${OAUTH_CLIENT_SECRET}
    prompt: select_account   # ← добавлено
    signIn:
      resolvers:
        - resolver: emailMatchingUserEntityProfileEmail
```

`select_account` — интерактивный prompt, позволяющий выбрать/ввести аккаунт без принудительной повторной аутентификации на каждый запрос. Authentik корректно обрабатывает его как "показать форму логина, если нет сессии; если есть — предложить выбор аккаунта".

Это **config-only изменение** — новый Docker-образ не требуется, достаточно перезапуска Backstage через ArgoCD.

---

## Итоговые изменения

### Файлы конфигурации

| Файл | Что изменено |
|------|-------------|
| `platform/backstage/helm-values.yaml` | `service.type: LoadBalancer`; `backend.csp.upgrade-insecure-requests: false`; image tag до `phase-4-task-4-6-r8` |
| `platform/backstage/app-config.auth.yaml` | `guest: null`; `app.signInPage: oidc`; `prompt: select_account`; добавлен resolver |
| `platform/backstage/app-config.catalog.yaml` | Добавлен `User` в allowed kinds |
| `catalog/all-components.yaml` | Добавлена User entity для `akadmin` (email `root@example.com`) |
| `platform/backstage/Dockerfile` | Патч backend (OIDC provider module); патч frontend (guest Component → OIDC popup, B loader → OIDC refresh, rename + html patch) |

### Docker образы

| Tag | Что изменено |
|-----|-------------|
| `phase-4-task-4-6-r1` | Исходный образ |
| `phase-4-task-4-6-r2` | Патч `providers:["oidc"]` → белый экран |
| `phase-4-task-4-6-r3` | Патч guest Component с popup (без postMessage) |
| `phase-4-task-4-6-r4` | Патч guest Component с popup + postMessage (nested payload) |
| `phase-4-task-4-6-r5` | Исправление postMessage payload (`d.response || d`) |
| `phase-4-task-4-6-r6` | Попытка cache bust через query param в index.html — не сработало |
| `phase-4-task-4-6-r7` | Rename + patch index.html — не сработало (app-backend игнорирует index.html) |
| `phase-4-task-4-6-r8` | Rename + patch index.html **и** index.html.tmpl — **работает** |

---

## Конфигурация /etc/hosts (браузер пользователя)

```
89.108.100.41   backstage.idp.local
```

Authentik доступен напрямую по IP: `http://89.108.100.218:80`
