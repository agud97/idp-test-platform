# Troubleshooting Matrix

## Назначение

Этот документ даёт быстрый путь от симптома к вероятной причине и командам проверки.

Формат:
- симптом
- вероятная причина
- куда смотреть
- чем проверить

## See Also

- [INDEX.md](/root/codex/idp-test/INDEX.md)
- [OPERATIONS_CHECKLIST.md](/root/codex/idp-test/OPERATIONS_CHECKLIST.md)
- [FAQ.md](/root/codex/idp-test/docs/FAQ.md)
- [TESTING_GUIDE.md](/root/codex/idp-test/docs/TESTING_GUIDE.md)

## Matrix

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
   - [OPERATIONS_CHECKLIST.md](/root/codex/idp-test/OPERATIONS_CHECKLIST.md) для более широкого runbook
   - [TESTING_GUIDE.md](/root/codex/idp-test/docs/TESTING_GUIDE.md) для полного workflow
