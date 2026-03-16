# Next Agent: Resolved Blockers And Remaining Risks

## Назначение

Этот документ коротко отвечает на практический вопрос:

если другому агенту дать текущий репозиторий, текущий план и новый Kubernetes
кластер, какие старые blockers уже сняты, а какие риски всё ещё останутся
внешними по отношению к репозиторию.

## Blockers, которые уже сняты

### 1. Пробел между `Environment` manifest и runtime provisioning

Раньше:
- в плане отсутствовал обязательный reconciliation layer между
  `kind: Environment` и `XWebappInstance` / `XPostgresqlInstance` / `XRedisInstance`

Сейчас:
- это закрыто через `phase-2.7`
- нужные задачи уже есть в [plan.yaml](plan.yaml)
- реализация уже есть в `platform/crds/`, `platform/crossplane/`,
  `platform/argocd/`

Итог:
- новый агент не должен заново открывать blocker вида
  “acceptance tests нельзя писать, пока нет provisioning path”

### 2. Несовпадение плана с реальной моделью ownership ArgoCD/Crossplane

Раньше:
- acceptance для self-heal была написана так, как будто ArgoCD напрямую владеет
  workload `Deployment` objects

Сейчас:
- `AC-037`, `AC-038`, `AC-040` и phase-5 task cards переписаны под реальную модель
- ArgoCD управляет `Environment` claim layer
- Crossplane управляет runtime resources

Итог:
- новый агент не должен снова спорить с планом на `task-5.3`

### 3. Нереалистичное ожидание `Degraded` для invalid manifest

Раньше:
- `AC-003` требовал `health=Degraded`
- в живом ArgoCD это не подтвердилось

Сейчас:
- `AC-003` и связанные task cards переписаны под наблюдаемое поведение:
  human-readable Argo sync failure + zero resources

Итог:
- новый агент не должен повторно упираться в этот acceptance mismatch

### 4. Backstage tasks были написаны как для source monorepo

Раньше:
- `task-4.3` / `4.4` / `4.5` / `4.6` предполагали normal Backstage source workspace
- в репозитории его нет; используется prebuilt image + runtime wiring

Сейчас:
- это зафиксировано в [constraints.md](constraints.md) и [plan.yaml](plan.yaml)
- task cards прямо допускают runtime integration under `platform/backstage/`

Итог:
- новый агент не должен снова открывать blocker вида
  “packages/app и packages/backend не участвуют в live build graph”

### 5. Scaffolder template discovery

Раньше:
- план не отражал, что template в `templates/` сам по себе не discoverable

Сейчас:
- это отражено в constraints и task scope
- Git-backed catalog registration теперь часть ожидаемого решения

Итог:
- новый агент не должен заново упираться в hidden template problem

### 6. Crossplane runtime prerequisites были неполно описаны

Раньше:
- план не включал все обязательные runtime prerequisites:
  - `provider-kubernetes`
  - Crossplane Functions
  - provider runtime RBAC
  - schema-alignment старых XRD

Сейчас:
- это отражено в `phase-2.7` task cards и constraints

Итог:
- новый агент не должен снова открывать blocker на `task-2.7.5` из-за
  отсутствующих Functions / RBAC / invalid XRD schema

### 7. Root-level Go module для repo-level tests

Раньше:
- root `go.mod` отсутствовал
- phase-5 tests из корня не запускались

Сейчас:
- root [go.mod](../go.mod) уже есть
- план и status больше не противоречат этому

Итог:
- новый агент не должен заново блокироваться на запуске acceptance/load tests

### 8. Load test observation path

Раньше:
- был риск ложной диагностики из-за неявного `KUBECONFIG` и неверного reconcile signal

Сейчас:
- `task-5.6` уточнён
- explicit `KUBECONFIG` и реальный reconcile path описаны в плане

Итог:
- новый агент не должен снова тратить время на ту же диагностику тестового harness

## Риски, которые всё ещё останутся для нового агента

### 1. Доступность Kubernetes API

Если API-сервер нестабилен:
- live validation может флапать
- acceptance/load tests могут давать ложные timeouts
- диагностика Argo/Crossplane станет ненадёжной

Это не проблема документации или кода, а проблема среды.

### 2. GitHub доступ и права на push

Новый агент всё ещё зависит от:
- рабочего `GITHUB_TOKEN`
- доступа на push в нужную ветку
- корректного remote URL

Если этого нет, GitOps validation не будет честной.

### 3. Pullable container images и registry reachability

Платформа зависит от того, что кластер может скачать:
- Backstage image
- Crossplane packages
- runtime app images
- provider/function packages

Если registry недоступен или возвращает ошибки, новый агент снова упрётся в runtime failures.

### 4. OIDC / Authentik availability

Backstage auth validation зависит от:
- живого Authentik
- корректной client configuration
- доступности DNS/service routing внутри кластера

Это не устраняется переписанным планом.

### 5. Cluster capacity для load test

`task-5.6` всё ещё требует реальный ресурсный минимум:
- 3 nodes
- 4 vCPU
- 8 GB RAM на node

Если новый кластер меньше или noisy, SLA assertions могут провалиться даже при корректном коде.

### 6. Network and DNS behavior

Acceptance на namespace isolation и runtime reachability зависят от:
- корректной работы CNI
- применения NetworkPolicy
- DNS resolution в кластере

Если это сломано в самом кластере, новый агент снова получит blockers уже на live validation.

## Практический вывод

Что теперь можно ожидать честно:
- новый агент не должен повторять старые plan/spec blockers
- новый агент должен пройти исполнение заметно прямее и без прежнего объёма пересогласований

Чего нельзя гарантировать:
- прохождение “без единого blocker вообще”

Причина:
- внешняя среда всё ещё важна не меньше, чем код и документы

## Что читать перед новым запуском

Минимальный набор:
- [plan.yaml](plan.yaml)
- [constraints.md](constraints.md)
- [acceptance_criteria.md](acceptance_criteria.md)
- [status.md](status.md)
- [POST_IMPLEMENTATION_DOC_CHANGES.md](POST_IMPLEMENTATION_DOC_CHANGES.md)

Практический контекст:
- [../HANDOFF.md](../HANDOFF.md)
- [../OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)
- [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)
