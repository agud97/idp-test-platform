# Practical Handoff

## See Also

- [INDEX.md](INDEX.md)
- [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md)
- [OPERATIONS_CHECKLIST.md](OPERATIONS_CHECKLIST.md)
- [TESTING_GUIDE.md](docs/TESTING_GUIDE.md)
- [USER_JOURNEYS.md](docs/USER_JOURNEYS.md)
- [FAQ.md](docs/FAQ.md)
- [docs/status.md](docs/status.md)

## Что запускать в первый день эксплуатации

1. Проверить состояние кластера и kubeconfig:

```bash
export KUBECONFIG=/root/codex/kubeconfig_6144665
kubectl get nodes
kubectl get pods -A
```

2. Проверить ArgoCD control plane и generated applications:

```bash
kubectl get applications.argoproj.io -n argocd
kubectl get applicationsets.argoproj.io -n argocd
```

3. Проверить Crossplane control plane:

```bash
kubectl get providers.pkg.crossplane.io
kubectl get providerconfigs
kubectl get functions.pkg.crossplane.io
kubectl get xrd,composition
```

4. Проверить Backstage:

```bash
kubectl get pods -n backstage
kubectl get svc -n backstage
# Backstage доступен на http://backstage.idp.local:30007
# /etc/hosts: 194.58.110.23 backstage.idp.local
```

5. Прогнать минимальный smoke:

```bash
go test ./tests/acceptance/... -run TestLifecycle -v
go test ./tests/security/... -v
```

## Какие секреты и переменные должны быть заведены

Git / GitHub:
- доступ на push в GitOps branch `feature/phase-1-foundation`
- token должен позволять:
  - clone
  - push
  - чтение branch tip для Argo/ApplicationSet

Kubernetes:
- рабочий kubeconfig с доступом как минимум к:
  - `argocd`
  - `crossplane-system`
  - environment namespaces `env-*`

Backstage:
- OIDC/Authentik настройки из:
  - [platform/backstage/app-config.auth.yaml](platform/backstage/app-config.auth.yaml)
- catalog/kubernetes/techdocs overrides из:
  - [platform/backstage/app-config.catalog.yaml](platform/backstage/app-config.catalog.yaml)
  - [platform/backstage/app-config.kubernetes.yaml](platform/backstage/app-config.kubernetes.yaml)
  - [platform/backstage/app-config.techdocs.yaml](platform/backstage/app-config.techdocs.yaml)

Load workflow:
- GitHub Actions secret:
  - `LOAD_TEST_KUBECONFIG_B64`

CLI migration tracker:
- для tracker DB при CLI lifecycle командах:
  - `IDP_TRACKER_DB`
- для deprecation reports:
  - `IDP_REPORT_DIR`

## Что мониторить после релиза

ArgoCD:
- generated `Application` objects для новых environments
- `status.sync.status`
- `status.health.status`
- stuck `OutOfSync` / repeated sync retries

Crossplane:
- `Environment` claims в `crossplane-system`
- `XEnvironment` readiness
- child XRs:
  - `XWebappInstance`
  - `XPostgresqlInstance`
  - `XRedisInstance`
- provider object errors и function resolution failures

Kubernetes:
- namespace creation/deletion latency для `env-*`
- stuck terminating namespaces
- pods в `CrashLoopBackOff` внутри environment namespaces
- PVC binding delays для postgres workloads

Backstage:
- login via OIDC
- catalog freshness
- scaffolder task failures
- `/migration` route and backend status endpoint

GitOps repo:
- unwanted leftover manifests after acceptance/load runs
- branch churn from test automation
- commits that fail to reconcile into generated Applications

## Canonical operational commands

Проверка конкретного environment:

```bash
kubectl get application -n argocd <team>-<env>
kubectl get environment -n crossplane-system <env>
kubectl get ns env-<team>-<env>
kubectl get all -n env-<team>-<env>
```

Проверка network policy baseline:

```bash
kubectl get netpol -n env-<team>-<env>
```

Проверка Backstage-driven template результата:

```bash
git log --oneline -- environments/<team>/<env>.yaml
kubectl get application -n argocd <team>-<env>
```

Проверка migration tracker:

```bash
IDP_TRACKER_DB=<path> ./cli/idp register-legacy --name <name> --team <team>
IDP_TRACKER_DB=<path> ./cli/idp complete-migration --env-id <id>
IDP_TRACKER_DB=<path> IDP_REPORT_DIR=<dir> ./cli/idp deprecate-legacy --dry-run
```

## Повторная валидация после изменений

После platform/GitOps changes:

```bash
go test ./tests/acceptance/... -run TestLifecycle -v
go test ./tests/acceptance/... -run TestGitOps -v
```

После CLI changes:

```bash
go test ./tests/acceptance/... -run TestMigration -v
```

После security/network changes:

```bash
go test ./tests/security/... -v
/tmp/kubescape scan platform/
```

Перед capacity/significant infra changes:

```bash
KUBECONFIG=/root/codex/kubeconfig_6144665 LOAD_TEST_COUNT=50 go test ./tests/load/... -run TestConcurrency -v -timeout 90m
```

## Осторожности при эксплуатации

- Acceptance и load tests делают реальные commits в GitOps branch. Их лучше запускать на выделенной ветке или в окне, где служебные commits допустимы.
- Если Kubernetes API начинает флапать по timeout, сначала проверять обычный `kubectl get nodes`, и только потом доверять acceptance/load результатам.
- Для `task-5.6` критично явно задавать правильный `KUBECONFIG`; без этого live-наблюдаемость может давать ложные пустые результаты.
- `kubescape` используется для `platform/`; проверка `environments/` на секреты идёт через [tests/security/credential_scan_test.go](tests/security/credential_scan_test.go), а не через scanner.

## Что передать следующему инженеру

- актуальный kubeconfig
- GitHub token/credential model для GitOps branch
- expected Argo app naming pattern:
  - `<team>-<env>`
- expected namespace naming pattern:
  - `env-<team>-<env>`
- canonical verification commands из этого файла
- ссылку на [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md)
