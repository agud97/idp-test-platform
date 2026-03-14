# Architecture Overview

## Назначение

Этот документ даёт краткий технический обзор архитектуры платформы:
- какие подсистемы есть
- как они связаны
- где проходит основной provisioning path
- какие точки управления и наблюдаемости считаются основными

## See Also

- [INDEX.md](../INDEX.md)
- [RELEASE_SUMMARY.md](../RELEASE_SUMMARY.md)
- [HANDOFF.md](../HANDOFF.md)
- [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)
- [status.md](status.md)

## High-Level Components

Платформа состоит из пяти основных частей:

1. GitOps source of truth
2. ArgoCD reconciliation layer
3. Crossplane provisioning layer
4. Backstage portal layer
5. CLI migration and legacy tracking layer

## GitOps Source of Truth

Основной пользовательский вход в платформу на уровне runtime — это Git.

Ключевые директории:
- [environments](../environments)
  - environment manifests
- [platform](../platform)
  - platform control manifests
- [catalog](../catalog)
  - Backstage catalog definitions
- [templates](../templates)
  - scaffolder templates

Ключевая идея:
- desired state хранится в Git
- платформа реагирует на Git commits, а не на ручные kubectl-изменения как на основной путь управления

## ArgoCD Layer

ArgoCD отвечает за:
- синхронизацию platform applications
- fan-out generated applications для `Environment` manifests
- self-heal на Argo-managed layer

Ключевые файлы:
- [app-platform.yaml](../platform/argocd/app-platform.yaml)
- [app-crossplane.yaml](../platform/argocd/app-crossplane.yaml)
- [app-backstage.yaml](../platform/argocd/app-backstage.yaml)
- [app-authentik.yaml](../platform/argocd/app-authentik.yaml)
- [appset-environments.yaml](../platform/argocd/appset-environments.yaml)

Generated Application naming:
- `<team>-<env>`

Пример:
- `environments/platform/demo.yaml`
- generated app `platform-demo`

Что Argo делает для environments:
- считывает `environments/**/*.yaml`
- создаёт generated `Application`
- применяет `Environment` claim в `crossplane-system`

## Crossplane Layer

Crossplane отвечает за превращение `Environment` claim в реальные runtime resources.

Основные элементы:
- XRD/claim:
  - [environment.yaml](../platform/crds/environment.yaml)
- environment composition:
  - [composition.yaml](../platform/crossplane/environment/composition.yaml)
- component compositions:
  - [webapp composition](../platform/crossplane/webapp/composition.yaml)
  - [postgresql composition](../platform/crossplane/postgresql/composition.yaml)
  - [redis composition](../platform/crossplane/redis/composition.yaml)
- providers/functions:
  - [provider-kubernetes.yaml](../platform/crossplane/provider-kubernetes.yaml)
  - [provider-kubernetes-rbac.yaml](../platform/crossplane/provider-kubernetes-rbac.yaml)
  - [functions.yaml](../platform/crossplane/functions.yaml)

### Provisioning Path

Основной путь выглядит так:

1. Git commit добавляет `environments/<team>/<env>.yaml`
2. `ApplicationSet` в ArgoCD генерирует `Application`
3. `Application` применяет `kind: Environment` в `crossplane-system`
4. Crossplane создаёт backing `XEnvironment`
5. `environment-composition` создаёт:
   - namespace `env-<team>-<env>`
   - baseline `NetworkPolicy`
   - child composites/resources по списку `spec.components`
6. component compositions создают runtime Kubernetes resources

### Naming

Claim:
- `<env>`

Namespace:
- `env-<team>-<env>`

### Ownership Model

Важно:
- ArgoCD владеет `Environment` manifest/claim layer
- Crossplane владеет runtime provisioning layer
- это означает, что direct drift на workload `Deployment` не всегда является Argo-owned drift

Именно поэтому acceptance tests фазы 5 были выровнены под реальную ownership model.

## Network Isolation

Каждый environment namespace получает baseline network policies:
- [allow-intra-namespace.yaml](../platform/networkpolicies/allow-intra-namespace.yaml)
- [deny-cross-namespace-ingress.yaml](../platform/networkpolicies/deny-cross-namespace-ingress.yaml)
- [allow-dns-egress.yaml](../platform/networkpolicies/allow-dns-egress.yaml)

Эта модель даёт:
- pod-to-pod связь внутри namespace
- блокировку cross-namespace ingress
- DNS egress в `kube-system`

## Backstage Layer

Backstage даёт пользователю portal interface для:
- login через OIDC
- просмотра catalog
- scaffolder template для нового environment
- migration dashboard
- TechDocs runbooks

Ключевые файлы:
- [app-config.auth.yaml](../platform/backstage/app-config.auth.yaml)
- [app-config.catalog.yaml](../platform/backstage/app-config.catalog.yaml)
- [app-config.kubernetes.yaml](../platform/backstage/app-config.kubernetes.yaml)
- [app-config.techdocs.yaml](../platform/backstage/app-config.techdocs.yaml)
- [template.yaml](../templates/new-environment/template.yaml)

Особенность реализации:
- Backstage в репозитории был интегрирован через prebuilt image / runtime wiring, а не через полноценный source monorepo

## CLI Migration Layer

CLI отвечает за:
- compose -> Environment conversion
- export legacy config
- manifest validation
- tracking migration lifecycle
- deprecation gating/reporting

Главный файл orchestration:
- [app.go](../cli/internal/cliapp/app.go)

Модули:
- [converter](../cli/pkg/converter)
- [exporter](../cli/pkg/exporter)
- [validator](../cli/pkg/validator)
- [tracker db](../cli/internal/tracker/db.go)

Команды:
- `migrate`
- `export`
- `validate`
- `register-legacy`
- `complete-migration`
- `deprecate-legacy`

## Testing and Validation Architecture

Validation разбита на три уровня:

### Acceptance

Файлы:
- [lifecycle_test.go](../tests/acceptance/lifecycle_test.go)
- [components_test.go](../tests/acceptance/components_test.go)
- [gitops_test.go](../tests/acceptance/gitops_test.go)
- [migration_test.go](../tests/acceptance/migration_test.go)

Что покрывают:
- lifecycle
- component behavior
- GitOps reconcile
- migration CLI behavior

### Security

Файлы:
- [credential_scan_test.go](../tests/security/credential_scan_test.go)
- [netpol_test.go](../tests/security/netpol_test.go)

Что покрывают:
- отсутствие literal credentials в `environments/`
- baseline network policies в live namespaces

### Load

Файл:
- [concurrency_test.go](../tests/load/concurrency_test.go)

Что покрывает:
- 50 environments
- sequential Git commits
- timing от commit до Argo-processed state
- cleanup и namespace deletion

Manual workflow:
- [load-test.yaml](../.github/workflows/load-test.yaml)

## CI / Automation

Основной pipeline:
- [ci.yaml](../.github/workflows/ci.yaml)

Он включает:
- build/lint
- CRD validation
- composition validation
- security scan
- security tests

## Operational Control Points

Наиболее важные точки наблюдения:

ArgoCD:
- generated `Application`
- sync/health/operationState

Crossplane:
- `Environment`
- `XEnvironment`
- child composites
- provider/function health

Kubernetes:
- namespace existence
- pods/services/pvc/netpol

Git:
- remote branch tip
- expected manifest path

Backstage:
- auth
- catalog
- scaffolder task results

## Known Architectural Constraints

- platform heavily depends on live Kubernetes API responsiveness
- acceptance/load tests делают реальные Git commits в рабочую branch history
- `kubectl`-driven live observation без явного `KUBECONFIG` может давать ложные отрицательные результаты
- `kubescape` meaningful for `platform/`, but not as the primary validator for custom manifests in `environments/`

## Practical Reading Order

Если нужен обзор:
1. [RELEASE_SUMMARY.md](../RELEASE_SUMMARY.md)
2. [HANDOFF.md](../HANDOFF.md)
3. [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)

Если нужно оперировать системой:
1. [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)
2. [HANDOFF.md](../HANDOFF.md)

Если нужно понять фактическую историю исполнения:
1. [status.md](status.md)
