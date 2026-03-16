# Post-Implementation Doc Changes

## Назначение

Этот документ фиксирует, какие нормативные документы были переписаны после
реального выполнения плана, и почему это понадобилось.

Цель:
- убрать расхождения между исходным планом и фактической реализацией
- снизить шанс того, что следующий агент снова упрётся в уже решённые blockers
- сделать execution history из [status.md](status.md) частью нормальной проектной документации

## Что было изменено

### 1. [acceptance_criteria.md](acceptance_criteria.md)

Изменения:
- переписан `AC-003`
- уточнён `AC-029`
- переписаны `AC-037`, `AC-038`, `AC-040`
- обновлены version/date

Почему:
- исходный `AC-003` требовал `Degraded` status для schema-invalid manifest
- в живом ArgoCD это поведение не проявилось как `health=Degraded`
- наблюдаемое и стабильно проверяемое поведение: human-readable sync failure + zero resources

- `AC-037`, `AC-038`, `AC-040` были написаны так, как будто ArgoCD напрямую владеет namespace workload resources
- фактическая архитектура этого репозитория другая:
  - ArgoCD управляет `Environment` claim layer
  - Crossplane управляет runtime resources
- поэтому acceptance semantics были приведены к claim-layer drift, а не к direct Deployment ownership

### 2. [constraints.md](constraints.md)

Изменения:
- добавлено явное правило про prebuilt-image Backstage deployment
- добавлено правило про обязательную Git-backed catalog registration для scaffolder templates
- обновлён minimal acceptance set для `AC-003`
- уточнён minimal acceptance set для `AC-037`

Почему:
- в репозитории нет полноценного live Backstage source monorepo
- без этого уточнения новый агент снова будет пытаться реализовать `packages/backend/...` и `packages/app/...` как будто они автоматически попадают в runtime
- это уже приводило к blockers на `task-4.4`, `task-4.5`, `task-4.6`

- также шаблон в `templates/` сам по себе не discoverable для running Backstage
- требовалась отдельная registration path через Git-backed catalog location

### 3. [plan.yaml](plan.yaml)

Изменения:
- `task-2.7.3` расширен до provider + Functions + runtime RBAC
- `task-2.7.4` теперь явно требует schema-alignment для existing XRDs перед GitOps sync
- `task-4.3` расширен: template registration и safe GitOps branch update wiring включены в scope
- `task-4.4` и `task-4.5` теперь прямо допускают runtime wiring через prebuilt image path
- `task-4.6` теперь прямо допускает equivalent runtime wiring для live `/docs`
- `task-5.1` переписан под фактическую семантику `AC-003`
- `task-5.3` переписан под реальную ownership model ArgoCD/Crossplane
- `task-5.5` уточнён: Kubescape сканирует `platform/`, а `environments/` покрывается credential audit
- `task-5.6` уточнён: explicit `KUBECONFIG` и reconcile signal измеряется по реальному provisioning path, а не по абстрактному одному только `Synced`

Почему:
- эти пункты были источниками реальных stop-and-discuss blockers во время исполнения
- после фактической реализации стало ясно, что исходные task cards были неполными для этого конкретного репозитория и его runtime architecture

## Что это даёт следующему агенту

После этих правок следующий агент:
- увидит `phase-2.7` как обязательную часть платформы, а не как ad-hoc rescue phase
- не будет предполагать, что Backstage собирается из source workspace
- не будет ожидать от `AC-003` недостижимого `Degraded` health
- не будет тестировать GitOps self-heal на неверном ownership layer
- не будет искать проблему в Kubescape для `environments/`, где основной контроль реализован иначе
- будет сразу знать, что load tests надо запускать с explicit `KUBECONFIG` и по реальному reconcile signal

## Что всё ещё остаётся внешней зависимостью

Даже после этих правок новый агент всё ещё зависит от:
- доступного Kubernetes API
- рабочего GitHub token / branch push access
- pullable container images
- рабочего OIDC/Authentiк setup
- live cluster capacity и DNS/network behavior

То есть правки документов убирают plan/spec blockers, но не устраняют runtime risks внешней среды.

## Связанные документы

- execution history: [status.md](status.md)
- итоговая архитектура: [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)
- release-level summary: [../RELEASE_SUMMARY.md](../RELEASE_SUMMARY.md)
- practical handoff: [../HANDOFF.md](../HANDOFF.md)
