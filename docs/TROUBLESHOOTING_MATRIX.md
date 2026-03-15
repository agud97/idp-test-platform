# Troubleshooting Matrix

## Назначение

Этот документ даёт быстрый путь от симптома к вероятной причине и командам проверки.

Формат:
- симптом
- вероятная причина
- куда смотреть
- чем проверить

## See Also

- [INDEX.md](../INDEX.md)
- [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)
- [FAQ.md](FAQ.md)
- [TESTING_GUIDE.md](TESTING_GUIDE.md)

## Matrix

### Симптом: Kubernetes-вкладка отсутствует на странице компонента

Две независимых причины — обе должны быть исправлены:

**Причина 1: `spec.type` не `service`**
- В prebuilt Backstage образе Kubernetes-вкладка рендерится только для `spec.type: service`
- Типы `environment`, `website`, и т.д. используют layout без Kubernetes-вкладки

**Причина 2: отсутствует аннотация `backstage.io/kubernetes-label-selector` (или `kubernetes-id`)**
- `isKubernetesAvailable` проверяет наличие `backstage.io/kubernetes-id` **или** `backstage.io/kubernetes-label-selector`
- Наличия только `kubernetes-cluster` и `kubernetes-namespace` недостаточно — вкладка не появится
- Это не задокументировано очевидно в Backstage docs; выявлено опытным путём

Диагностика:
```bash
# Проверить тип entity
curl -s "http://89.108.100.41:7007/api/catalog/entities?filter=kind=component,metadata.name=<name>" | \
  python3 -c "import sys,json; e=json.load(sys.stdin); print(e[0]['spec']['type'], e[0]['metadata'].get('annotations',{}).get('backstage.io/kubernetes-label-selector','MISSING')) if e else print('not found')"
# Ожидаемо: service  idp.platform.io/team
```

Исправление:
1. Изменить `spec.type: environment` → `spec.type: service`
2. Добавить аннотацию `backstage.io/kubernetes-label-selector: idp.platform.io/team`
3. Применить изменения в **обоих** местах: catalog entry файл + `templates/new-environment/skeleton/catalog-info.yaml`
4. Пересобрать образ (catalog файлы embedded в Docker image)

---

### Симптом: Backstage показывает "Enter as a Guest User" вместо OIDC

Вероятные причины:
- `guest.Component` в prebuilt образе не заменён OIDC-патчем
- браузер отдаёт старый `module-backstage.*.js` из кэша (max-age 2 недели)
- `index.html.tmpl` не патчен — Backstage сервирует шаблон, а не `index.html`

Куда смотреть:
- HTML из которого браузер загружает страницу
- имя файла `module-backstage.*.js` в HTML
- `Cache-Control` заголовок на JS-файле

Команды:
```bash
# Проверить, что сервер отдаёт правильный JS-файл
curl -s http://89.108.100.41:7007 | grep 'module-backstage'
# Ожидаемо: module-backstage.oidcpatch.js

# Проверить, что патч применён
kubectl exec deployment/backstage -n backstage -- \
  grep -c 'oidc/refresh' /app/packages/app/dist/static/module-backstage.oidcpatch.js
# Ожидаемо: 1

# Проверить что index.html.tmpl указывает на патченый файл
kubectl exec deployment/backstage -n backstage -- \
  grep 'module-backstage' /app/packages/app/dist/index.html.tmpl
```

Важно: `index.html` в этом образе **не используется** — app-backend регенерирует HTML из `index.html.tmpl` при каждом запросе. Патчить нужно именно `index.html.tmpl`.

---

### Симптом: OIDC логин проходит, но Backstage показывает "Failed to load user identity: TypeError: this.config.identityApi.getProfileInfo is not a function"

Вероятная причина:
- Патч передаёт `getProfile` вместо `getProfileInfo` в объект identity
- Отсутствует метод `getCredentials` в объекте identity

Backstage `IdentityApi` требует точно: `getBackstageIdentity`, `getProfileInfo`, `getCredentials`, `signOut`.

Команды:
```bash
# Проверить что патч содержит getProfileInfo (должно быть 3)
kubectl exec deployment/backstage -n backstage -- \
  grep -c 'getProfileInfo' /app/packages/app/dist/static/module-backstage.oidcpatch.js
# Ожидаемо: 3

# Проверить что getProfile (старое неверное имя) отсутствует
kubectl exec deployment/backstage -n backstage -- \
  grep -c '"getProfile"' /app/packages/app/dist/static/module-backstage.oidcpatch.js
# Ожидаемо: 0
```

Исправление: пересобрать образ с исправленным патчем — заменить `getProfile` → `getProfileInfo`, добавить `getCredentials: async () => ({token: payload.backstageIdentity && payload.backstageIdentity.token})` во всех трёх местах.

---

### Симптом: OIDC popup открывается и сразу закрывается (~300ms), ошибка `login_required`

Вероятная причина:
- Backstage отправляет `prompt=none` по умолчанию
- Authentik возвращает `login_required` немедленно, если у пользователя нет активной сессии

Команды:
```bash
# Проверить какой prompt идёт в запросе к Authentik
curl -sv "http://89.108.100.41:7007/api/auth/oidc/start?env=production" 2>&1 | grep Location
# Если в Location видно &prompt=none& → проблема подтверждена

kubectl logs -n backstage deployment/backstage --tail=20
# Ищем: handler/frame?error=login_required
```

Исправление: добавить `prompt: select_account` в `platform/backstage/app-config.auth.yaml`:
```yaml
oidc:
  production:
    prompt: select_account
```

Не использовать `prompt: login` — вызывает бесконечный цикл переаутентификации в Authentik.

---

### Симптом: OIDC popup проходит, но пользователь не попадает в Backstage, ошибка "could not find user entity"

Вероятные причины:
- User entity для пользователя отсутствует в Backstage catalog
- Resolver `emailMatchingUserEntityProfileEmail` не может найти User по email из OIDC токена
- `User` kind не разрешён в catalog location rules

Команды:
```bash
kubectl logs -n backstage deployment/backstage --tail=30 | grep -i 'resolver\|sign.in\|email\|user entity'

# Проверить, что User entity есть в catalog
curl -s "http://89.108.100.41:7007/api/catalog/entities?filter=kind=user" | python3 -m json.tool | grep -A5 '"name"'
```

Исправление:
1. Добавить User entity в `catalog/all-components.yaml` (email должен совпадать с email в Authentik)
2. Добавить `User` в allowed kinds в `platform/backstage/app-config.catalog.yaml`

---

### Симптом: `kubectl` зависает или даёт `context deadline exceeded`

Вероятные причины:
- нестабильный Kubernetes API
- network path issue
- неправильный `KUBECONFIG`

Куда смотреть:
- cluster readiness
- local env

Команды:

```bash
echo "$KUBECONFIG"
kubectl get nodes
kubectl get --raw=/readyz
```

### Симптом: manifest уже в Git, но generated Argo Application не появился

Вероятные причины:
- файл не попал в watched branch
- ApplicationSet не пересобрал applications
- ArgoCD не видит repo change

Куда смотреть:
- Git branch
- `ApplicationSet`
- generated apps list

Команды:

```bash
git ls-remote --heads origin feature/phase-1-foundation
kubectl get applicationsets.argoproj.io -n argocd
kubectl get applications.argoproj.io -n argocd
```

### Симптом: generated Application есть, но `OutOfSync`

Вероятные причины:
- invalid manifest
- sync failure
- controller ещё не довёл ресурс до steady state

Куда смотреть:
- `Application.status.operationState`
- syncResult resource messages

Команды:

```bash
kubectl get application -n argocd <team>-<env> -o yaml
```

### Симптом: generated Application `Synced`, но `Environment` claim отсутствует

Вероятные причины:
- apply path не создал claim
- resource rejected on apply
- target path/namespace mismatch

Куда смотреть:
- `Application.status.resources`
- syncResult messages

Команды:

```bash
kubectl get application -n argocd <team>-<env> -o yaml
kubectl get environment -n crossplane-system <env> -o yaml
```

### Симптом: `Environment` claim есть, но `Ready=False`

Вероятные причины:
- Crossplane composition issue
- missing provider/function
- child resource reconcile failure

Куда смотреть:
- claim conditions
- backing `XEnvironment`
- Crossplane packages

Команды:

```bash
kubectl get environment -n crossplane-system <env> -o yaml
kubectl get xenvironment -n crossplane-system -o wide
kubectl get providers.pkg.crossplane.io
kubectl get functions.pkg.crossplane.io
kubectl get providerconfigs
```

### Симптом: namespace не создаётся

Вероятные причины:
- environment composition failure
- provider-kubernetes issue
- claim not actually reconciled

Куда смотреть:
- `Environment`
- `XEnvironment`
- provider config / provider health

Команды:

```bash
kubectl get environment -n crossplane-system <env> -o yaml
kubectl get xenvironment -n crossplane-system -o wide
kubectl get ns env-<team>-<env>
```

### Симптом: namespace есть, но web/database resources отсутствуют

Вероятные причины:
- component disabled
- child composition failure
- provider object creation failure

Куда смотреть:
- `spec.components`
- runtime resources
- child composites

Команды:

```bash
kubectl get environment -n crossplane-system <env> -o yaml
kubectl get all -n env-<team>-<env>
kubectl get configmap,secret,pvc -n env-<team>-<env>
```

### Симптом: pod в `CrashLoopBackOff`

Вероятные причины:
- bad image
- missing config
- missing secret
- app-level startup failure

Куда смотреть:
- pod describe
- logs
- ConfigMap/Secret references

Команды:

```bash
kubectl describe pod -n env-<team>-<env> <pod>
kubectl logs -n env-<team>-<env> <pod>
kubectl get configmap,secret -n env-<team>-<env>
```

### Симптом: database не поднимается

Вероятные причины:
- pvc not bound
- secret/config issue
- postgres composition failure

Куда смотреть:
- PVC
- Secret
- runtime resources

Команды:

```bash
kubectl get pvc -n env-<team>-<env>
kubectl get secret -n env-<team>-<env>
kubectl get all -n env-<team>-<env>
```

### Симптом: cross-namespace traffic не блокируется

Вероятные причины:
- baseline `NetworkPolicy` missing
- probe выполнен не pod-to-pod, а через неверный path
- namespace не тот

Куда смотреть:
- policies
- actual source/target namespaces

Команды:

```bash
kubectl get netpol -n env-<team>-<env>
kubectl get netpol -n env-<other-team>-<other-env>
```

### Симптом: intra-namespace traffic тоже не работает

Вероятные причины:
- service/port mismatch
- probe на wrong target
- pod не ready

Куда смотреть:
- service
- pod labels
- ports

Команды:

```bash
kubectl get svc -n env-<team>-<env>
kubectl get pod -n env-<team>-<env> -o wide
kubectl get deployment -n env-<team>-<env> -o yaml
```

### Симптом: `idp migrate` создал плохой manifest

Вероятные причины:
- compose содержит unsupported services/features
- service type не мапится
- manual review warnings были проигнорированы

Куда смотреть:
- conversion report
- output manifest

Команды:

```bash
./cli/idp migrate --from ./docker-compose.yml --output ./environment.yaml
sed -n '1,220p' ./environment.yaml
./cli/idp validate --file ./environment.yaml
```

### Симптом: unknown service оказался `enabled: false`

Вероятная причина:
- это штатное поведение converter для неподдерживаемого типа

Что делать:
- не включать его вслепую
- сначала определить, можно ли мапить в существующий type
- если нет, проектировать новый platform component type

### Симптом: `complete-migration` выдаёт ошибку про статус

Вероятная причина:
- legacy environment ещё не `validated`

Что делать:
- проверить tracker state
- перевести запись в `validated`, если бизнес-процесс это разрешает

### Симптом: `deprecate-legacy --dry-run` блокируется

Вероятная причина:
- хотя бы один legacy environment ещё не `completed`

Куда смотреть:
- tracker summary
- blocking records

Команды:

```bash
IDP_TRACKER_DB=<db> ./cli/idp deprecate-legacy --dry-run
```

### Симптом: deprecation report не создаётся

Вероятные причины:
- не задан `IDP_REPORT_DIR`
- gate не прошёл
- запущен `--dry-run` вместо `--confirm`

Команды:

```bash
export IDP_REPORT_DIR=./reports
IDP_TRACKER_DB=<db> ./cli/idp deprecate-legacy --confirm
ls -1 ./reports/deprecation-report-*.md
```

### Симптом: `go test ./tests/load/...` зависает

Вероятные причины:
- неправильный `KUBECONFIG`
- API flapping
- teardown wait
- GitOps branch commits не доходят до remote

Куда смотреть:
- branch tip
- environment manifests under `environments/load`
- `env-load-*` namespaces

Команды:

```bash
git ls-remote --heads origin feature/phase-1-foundation
kubectl get ns | grep '^env-load-'
```

### Симптом: load test timings хуже SLA

Вероятные причины:
- cluster capacity degradation
- Argo reconciliation slowdown
- Crossplane provisioning slowdown
- API slowness

Что делать:
- сравнить current timing table с baseline
- проверить cluster/system namespaces

Команды:

```bash
kubectl get nodes
kubectl get pods -n argocd
kubectl get pods -n crossplane-system
```

### Симптом: security test на literal credentials падает

Вероятная причина:
- в `environments/` попало что-то вроде:
  - `password: value`
  - `secret: value`
  - `token: value`
  - `privateKey: value`

Что делать:
- убрать literal secret
- заменить на безопасную ссылку/managed secret pattern

Команда:

```bash
go test ./tests/security/... -v
```

### Симптом: `kubescape scan platform/` не проходит

Вероятные причины:
- появились новые findings
- scanner binary отсутствует
- CI/security baseline изменился

Что делать:
- убедиться, что используется корректный binary
- посмотреть failing control details

Команды:

```bash
/tmp/kubescape scan platform/
/tmp/kubescape scan control <CONTROL-ID> platform/ -v
```

## Как использовать этот документ

Рекомендуемый путь:

1. выбрать симптом
2. выполнить 1-2 команды из соответствующей секции
3. определить, на каком слое проблема:
   - Git
   - ArgoCD
   - Crossplane
   - Kubernetes runtime
   - CLI / tracker
4. затем перейти в:
   - [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md) для более широкого runbook
   - [TESTING_GUIDE.md](TESTING_GUIDE.md) для полного workflow
