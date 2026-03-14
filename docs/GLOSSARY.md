# Glossary

## Назначение

Этот документ собирает основные термины платформы в одном месте, чтобы снизить путаницу между GitOps, ArgoCD, Crossplane, Backstage и CLI слоями.

## See Also

- [INDEX.md](/root/codex/idp-test/INDEX.md)
- [ARCHITECTURE_OVERVIEW.md](/root/codex/idp-test/docs/ARCHITECTURE_OVERVIEW.md)
- [TESTING_GUIDE.md](/root/codex/idp-test/docs/TESTING_GUIDE.md)
- [FAQ.md](/root/codex/idp-test/docs/FAQ.md)

## Terms

### Environment

Пользовательский namespaced claim типа `idp.platform.io/v1alpha1`.

Где находится:
- [platform/crds/environment.yaml](/root/codex/idp-test/platform/crds/environment.yaml)

Что означает:
- описывает test environment на уровне desired state
- содержит `spec.team`, `spec.owner`, `spec.components[]`

### XEnvironment

Backing Crossplane composite resource для `Environment`.

Что означает:
- внутренний composite resource, который создаётся за claim'ом
- именно он управляется composition logic

### Component

Один элемент в `Environment.spec.components[]`.

Что означает:
- отдельный сервис или часть environment
- имеет `name`, `type`, `enabled`, и, в зависимости от типа, дополнительные параметры

### Component Type

Логический тип компонента платформы.

Поддерживаемые текущие типы:
- `webapp`
- `postgresql`
- `redis`

### GitOps Branch

Git branch, за которым наблюдает ArgoCD для environment manifests.

Текущий branch:
- `feature/phase-1-foundation`

### Generated Application

ArgoCD `Application`, автоматически созданный `ApplicationSet` для конкретного environment manifest.

Шаблон имени:
- `<team>-<env>`

Пример:
- `platform-demo`

### ApplicationSet

ArgoCD ресурс, который сканирует Git и генерирует `Application` objects.

Где находится:
- [platform/argocd/appset-environments.yaml](/root/codex/idp-test/platform/argocd/appset-environments.yaml)

### Source of Truth

Источник истинного desired state.

В этой платформе:
- Git

### Self-Heal

Способность ArgoCD вернуть Argo-managed resources к состоянию из Git.

Важно:
- self-heal здесь в первую очередь относится к `Environment`/generated `Application` layer
- runtime resources ниже по стеку создаются Crossplane/provider layer

### Crossplane Composition

Шаблон, который описывает, какие ресурсы должны создаваться из composite resource.

Примеры:
- [platform/crossplane/environment/composition.yaml](/root/codex/idp-test/platform/crossplane/environment/composition.yaml)
- [platform/crossplane/webapp/composition.yaml](/root/codex/idp-test/platform/crossplane/webapp/composition.yaml)

### Provider

Crossplane package, которое умеет создавать ресурсы во внешней системе или в Kubernetes.

Здесь ключевой provider:
- `provider-kubernetes`

### ProviderConfig

Crossplane configuration object, через который provider знает, куда и как применять ресурсы.

Здесь ключевой объект:
- `kubernetes-provider`

### Function

Crossplane function package, используемый pipeline compositions.

Здесь используются function packages для templating/ready logic.

### Namespace Isolation

Модель, при которой каждый environment живёт в отдельном namespace, а cross-namespace трафик ограничен baseline `NetworkPolicy`.

### Baseline Network Policies

Три обязательные политики для каждого environment namespace:
- allow intra-namespace traffic
- deny cross-namespace ingress
- allow DNS egress

### Backstage Scaffolder

Portal workflow, который рендерит template и делает Git commit нового environment manifest.

### Migration Tracker

CLI-backed tracker DB для legacy environments и migration lifecycle.

Ключевые статусы:
- `pending`
- `validated`
- `completed`
- другие промежуточные/служебные статусы tracker'а

### Deprecation Report

Markdown report, который генерируется при успешном `deprecate-legacy --confirm`.

Что содержит:
- environment IDs
- team
- environment name
- completed timestamp

### Acceptance Test

Live test, который проверяет platform behavior через реальный GitOps path, а не через isolated mock-only execution.

### Security Test

Тест на отсутствие literal credentials и на наличие baseline network policies.

### Load Test

Тест, который создаёт много environments и измеряет timing до reconcile state.

### KUBECONFIG

Переменная окружения, определяющая, к какому кластеру ходит `kubectl`.

Canonical value для этого репозитория:

```bash
/root/codex/kubeconfig_6144665
```

### Runtime Resources

Kubernetes resources, которые реально живут в environment namespace:
- Deployment
- Service
- ConfigMap
- Secret
- PVC
- NetworkPolicy

### Legacy Application

Существующее приложение вне новой платформенной модели, обычно описанное через `docker-compose` и/или старый runtime path.
