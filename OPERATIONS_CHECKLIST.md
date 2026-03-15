# Operations Checklist

## See Also

- [INDEX.md](INDEX.md)
- [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md)
- [HANDOFF.md](HANDOFF.md)
- [TESTING_GUIDE.md](docs/TESTING_GUIDE.md)
- [USER_JOURNEYS.md](docs/USER_JOURNEYS.md)
- [FAQ.md](docs/FAQ.md)
- [docs/status.md](docs/status.md)

## Назначение

Этот файл предназначен для повседневной эксплуатации платформы:
- ежедневные проверки здоровья
- диагностика конкретного environment
- безопасный повторный запуск валидаций
- действия перед и после изменений

Он не заменяет [HANDOFF.md](HANDOFF.md) и [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md), а дополняет их более прикладным операционным списком.

## Базовая подготовка

Перед любыми действиями в этой репозитории:

```bash
cd /root/codex/idp-test
export KUBECONFIG=/root/codex/kubeconfig_6144665
```

Пояснение:
- для live acceptance, load и debug-команд нужен именно этот kubeconfig
- без него часть проверок может давать ложные `NotFound`, пустые `jsonpath` или timeout-поведение

## Ежедневный Health Check

### 1. Проверить Kubernetes API и ноды

```bash
kubectl get nodes
kubectl get --raw=/readyz
```

Что считать нормой:
- все ноды `Ready`
- `/readyz` возвращает `ok`

Что считать проблемой:
- хотя бы одна нода не `Ready`
- `kubectl` зависает или даёт `context deadline exceeded`

Если проблема есть:
- не доверять acceptance/load результатам до стабилизации API
- сначала диагностировать control plane / network path

### 2. Проверить системные namespace

```bash
kubectl get pods -n argocd
kubectl get pods -n crossplane-system
kubectl get pods -n backstage
```

Что считать нормой:
- нет `CrashLoopBackOff`
- нет массовых `ImagePullBackOff`
- нет pod'ов, зависших в `Pending`

Что важно:
- `argocd` отвечает за GitOps reconciliation
- `crossplane-system` отвечает за composite provisioning
- `backstage` отвечает за portal/scaffolder/dashboard path

### 3. Проверить ArgoCD Applications и ApplicationSet

```bash
kubectl get applications.argoproj.io -n argocd
kubectl get applicationsets.argoproj.io -n argocd
```

Что считать нормой:
- platform apps в `Synced` / `Healthy`
- generated environment applications присутствуют для активных manifests

Что считать проблемой:
- persistent `OutOfSync`
- `Degraded`
- `Missing`
- generated app не появляется после нового manifest commit

## Проверка Crossplane

### 4. Проверить provider / function / XRD / Composition

```bash
kubectl get providers.pkg.crossplane.io
kubectl get providerconfigs
kubectl get functions.pkg.crossplane.io
kubectl get xrd
kubectl get composition
```

Что считать нормой:
- provider packages `Healthy`
- `kubernetes-provider` существует
- required functions установлены
- XRD и Composition ресурсы присутствуют

Что считать проблемой:
- нет `providerConfig`
- нет `FunctionRevision`
- Composition отсутствует или не применяется

### 5. Проверить Environment claims

```bash
kubectl get environment -n crossplane-system
kubectl get xenvironment -n crossplane-system
```

Что считать нормой:
- активные claims имеют `Synced=True`
- в steady state `Ready=True`

Что считать проблемой:
- claim создан, но `Ready=False` длительное время
- claim есть, но backing `XEnvironment` не создаётся

Для детальной проверки конкретного environment:

```bash
kubectl get environment -n crossplane-system <env> -o yaml
kubectl get xenvironment -n crossplane-system -o wide
```

Смотреть на:
- `.status.conditions`
- `.spec.resourceRef.name`
- ошибки reconcile/message

## Backstage OIDC — первый деплой / повторный деплой

При первом деплое или пересборке образа Backstage нужно проверить следующие пункты:

### Backstage-0. Проверить сервис

```bash
kubectl get svc -n backstage
# Ожидаемо: type=LoadBalancer, EXTERNAL-IP=89.108.100.41, PORT=7007
```

### Backstage-1. Проверить, что патченый JS сервируется

```bash
curl -s http://89.108.100.41:7007 | grep 'module-backstage'
# Ожидаемо: module-backstage.oidcpatch.js
```

Если видно `module-backstage.3be92c1c.js` (или другой hash) — патч Dockerfile не применён или ArgoCD синхронизировал старый образ.

### Backstage-2. Проверить, что patch применён к JS

```bash
kubectl exec deployment/backstage -n backstage -- \
  grep -c 'oidc/refresh' /app/packages/app/dist/static/module-backstage.oidcpatch.js
# Ожидаемо: 1

kubectl exec deployment/backstage -n backstage -- \
  grep -c 'enableLegacyGuestToken' /app/packages/app/dist/static/module-backstage.oidcpatch.js
# Ожидаемо: 0
```

### Backstage-3. Проверить OIDC start redirect содержит нужный prompt

```bash
curl -sv "http://89.108.100.41:7007/api/auth/oidc/start?env=production" 2>&1 | grep -o 'prompt=[^&]*'
# Ожидаемо: prompt=select_account
# НЕ должно быть: prompt=none (users will get login_required), prompt=login (infinite loop)
```

### Backstage-4. Проверить User entity в catalog

```bash
curl -s "http://89.108.100.41:7007/api/catalog/entities?filter=kind=user,metadata.name=akadmin" | \
  python3 -c "import sys,json; d=json.load(sys.stdin); print('OK' if d else 'MISSING')"
# Ожидаемо: OK
```

Если `MISSING` — добавить User entity в `catalog/all-components.yaml` (email должен совпадать с email пользователя в Authentik).

### Backstage-5b. Проверить spec.type в catalog entries

```bash
curl -s "http://89.108.100.41:7007/api/catalog/entities?filter=kind=component,spec.type=service" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d), 'service-type components')"
# Все environment entries должны быть type=service, иначе Kubernetes-вкладка не появится
```

### Backstage-5. Проверить end-to-end логин

Открыть `http://backstage.idp.local:7007` в браузере (с `/etc/hosts` `89.108.100.41 backstage.idp.local` и `89.108.100.218 authentik-server.authentik.svc.cluster.local`), нажать "Sign in with OIDC", залогиниться как `akadmin` / `Admin1234!`.

**Критичные настройки /etc/hosts в Backstage при повторном развёртывании:**

| Запись | IP | Назначение |
|--------|-----|-----------|
| `backstage.idp.local` | `89.108.100.41` | Backstage portal |
| `authentik-server.authentik.svc.cluster.local` | `89.108.100.218` | Authentik OIDC (браузер редиректится на internal DNS-имя) |

---

## Проверка конкретного Environment

Ниже шаблон для диагностики одного environment.

Допустим:
- team = `platform`
- env = `smoke-template-0314h`

### 6. Проверить generated Argo Application

```bash
kubectl get application -n argocd platform-smoke-template-0314h -o yaml
```

Смотреть на:
- `.status.sync.status`
- `.status.health.status`
- `.status.operationState.phase`
- `.status.operationState.message`

Интерпретация:
- `Synced` + `Healthy` = Argo свою часть выполнил
- если дальше workload не готов, смотреть уже в Crossplane / namespace resources

### 7. Проверить claim и namespace

```bash
kubectl get environment -n crossplane-system smoke-template-0314h -o yaml
kubectl get ns env-platform-smoke-template-0314h -o yaml
```

Что это даёт:
- подтверждает, что Git manifest материализовался в claim
- подтверждает, что namespace создан composition path'ом

### 8. Проверить runtime ресурсы в namespace

```bash
kubectl get all -n env-platform-smoke-template-0314h
kubectl get configmap,secret,pvc -n env-platform-smoke-template-0314h
kubectl get netpol -n env-platform-smoke-template-0314h
```

Что считать нормой:
- web/database/cache появляются в соответствии с manifest
- baseline network policies присутствуют
- PVC bound, если есть postgresql

## Проверка GitOps path

### 9. Проверить, что manifest реально в Git branch

```bash
git rev-parse --abbrev-ref HEAD
git remote get-url origin
git ls-remote --heads origin feature/phase-1-foundation
```

Если нужно проверить конкретный файл на remote:

```bash
tmp=$(mktemp -d)
git clone --depth 1 --branch feature/phase-1-foundation "$(git remote get-url origin)" "$tmp"
find "$tmp/environments" -maxdepth 3 -type f | sort
rm -rf "$tmp"
```

Зачем это нужно:
- отделяет Git push issue от Argo/ApplicationSet issue
- если файла нет в remote branch, бесполезно искать проблему в кластере

### 10. Проверить naming conventions

Generated Argo Application:
- `<team>-<env>`

Namespace:
- `env-<team>-<env>`

Пример:
- manifest `environments/platform/demo.yaml`
- generated app `platform-demo`
- namespace `env-platform-demo`

Если ищешь не то имя:
- можно получить ложный `NotFound`
- особенно важно для acceptance/load test диагностики

## Проверка Backstage

### 11. Проверить базовую доступность Backstage

```bash
kubectl get pods -n backstage
kubectl get svc -n backstage
# LoadBalancer → http://backstage.idp.local:7007
# /etc/hosts: 89.108.100.41 backstage.idp.local
```

Для доступа из браузера добавить в `/etc/hosts`:
```
89.108.100.41   backstage.idp.local
89.108.100.218  authentik-server.authentik.svc.cluster.local
```

Проверить:
- login через OIDC
- catalog entities
- scaffolder template
- migration dashboard
- docs/runbooks

### 12. Проверить scaffolder результат

После запуска шаблона:

```bash
git log --oneline -- environments/<team>/<env>.yaml
kubectl get application -n argocd <team>-<env>
kubectl get environment -n crossplane-system <env>
```

Это подтверждает:
- template сделал Git commit
- Argo создал generated app
- environment claim появился

## Проверка Migration CLI

### 13. Проверить tracker DB path

```bash
echo "$IDP_TRACKER_DB"
```

Если переменная не задана:
- CLI будет использовать default path
- это может вести к чтению/записи не той базы, что ты ожидаешь

### 14. Базовые CLI lifecycle команды

```bash
IDP_TRACKER_DB=<db-path> ./cli/idp register-legacy --name <name> --team <team>
IDP_TRACKER_DB=<db-path> ./cli/idp complete-migration --env-id <id>
IDP_TRACKER_DB=<db-path> IDP_REPORT_DIR=<dir> ./cli/idp deprecate-legacy --dry-run
```

Что проверять:
- ошибки team lookup
- duplicate registration
- gate blocked/pass
- report generation

### 15. Проверить deprecation report

Файл будет вида:

```bash
ls -1 <report-dir>/deprecation-report-*.md
```

Внутри должны быть:
- `Environment ID`
- `Team`
- `Environment`
- `Completed At`
- `Target Environment ID`

## Проверка Security

### 16. Проверить credential audit

```bash
go test ./tests/security/... -v
```

Что покрывается:
- отсутствие literal credential-like values в `environments/`
- наличие baseline network policies в live env namespaces

### 17. Проверить Kubescape

```bash
/tmp/kubescape scan platform/
```

Что считать нормой:
- zero `CRITICAL`

Важно:
- scanner используется как meaningful check для `platform/`
- `environments/` проверяются на секреты отдельным Go test, а не scanner'ом

## Повторная валидация после изменений

### 18. После изменения platform manifests

```bash
go test ./tests/acceptance/... -run TestLifecycle -v
go test ./tests/acceptance/... -run TestGitOps -v
go test ./tests/security/... -v
```

### 19. После изменения component/database compositions

```bash
go test ./tests/acceptance/... -run TestComponents -v
```

### 20. После изменения CLI

```bash
go test ./tests/acceptance/... -run TestMigration -v
```

### 21. После изменения capacity/GitOps/performance-sensitive parts

```bash
KUBECONFIG=/root/codex/kubeconfig_6144665 LOAD_TEST_COUNT=50 go test ./tests/load/... -run TestConcurrency -v -timeout 90m
```

Интерпретация:
- timing table печатается в output
- все environments должны уложиться в SLA
- cleanup должен удалить `env-load-*`

## Операционный Runbook для проблем

### Сценарий A: Git manifest есть, generated app не появился

Проверки:

```bash
git ls-remote --heads origin feature/phase-1-foundation
kubectl get applicationsets.argoproj.io -n argocd environments -o yaml
kubectl get applications.argoproj.io -n argocd
```

Скорее всего проблема в:
- ApplicationSet reconcile
- repo polling
- controller errors в ArgoCD

### Сценарий B: Application `Synced`, но namespace/workload нет

Проверки:

```bash
kubectl get application -n argocd <team>-<env> -o yaml
kubectl get environment -n crossplane-system <env> -o yaml
kubectl get xenvironment -n crossplane-system -o wide
```

Скорее всего проблема в:
- Crossplane composition
- function/provider path
- downstream object reconcile

### Сценарий C: Namespace есть, workload не healthy

Проверки:

```bash
kubectl get all -n env-<team>-<env>
kubectl describe pod -n env-<team>-<env> <pod>
kubectl logs -n env-<team>-<env> <pod>
```

Скорее всего проблема в:
- image pull
- runtime misconfig
- PVC / secret / config dependency

### Сценарий D: Acceptance/load tests дают пустые результаты

Проверки:

```bash
echo "$KUBECONFIG"
kubectl get nodes
```

Важно:
- сначала проверить kubeconfig
- потом API responsiveness
- только потом доверять test failures

Это уже было реальной причиной ложных сбоев в live-наблюдаемости.

## Чего не делать

- не запускать acceptance/load tests без понимания, что они делают реальные Git commits
- не интерпретировать `kubectl` timeout как platform bug без проверки API stability
- не искать generated app по неправильному имени
- не полагаться только на Argo `Healthy` для оценки Crossplane-ready состояния

## Минимальный рабочий набор команд

Если нужен самый короткий practical path:

```bash
export KUBECONFIG=/root/codex/kubeconfig_6144665
kubectl get nodes
kubectl get applications.argoproj.io -n argocd
kubectl get environment -n crossplane-system
kubectl get ns | grep '^env-'
go test ./tests/security/... -v
go test ./tests/acceptance/... -run TestLifecycle -v
```

Если это всё зелёное:
- платформа обычно в хорошем операционном состоянии

## Связанные документы

- [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md)
- [HANDOFF.md](HANDOFF.md)
- [docs/status.md](docs/status.md)
