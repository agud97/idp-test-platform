# Testing Guide

## Назначение

Этот документ — практический guide по использованию платформы:
- как запускать и понимать тесты
- как мигрировать legacy приложения из `docker-compose`
- как создавать и запускать тестовые environments
- как добавлять новые сервисы и component types

Документ рассчитан на инженера, который работает с этим репозиторием впервые и хочет не только запускать команды, но и понимать, что именно происходит.

## See Also

- [INDEX.md](../INDEX.md)
- [RELEASE_SUMMARY.md](../RELEASE_SUMMARY.md)
- [HANDOFF.md](../HANDOFF.md)
- [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)
- [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)
- [status.md](status.md)

## 1. Базовая подготовка

Перед любыми live-действиями:

```bash
cd /root/codex/idp-test
export KUBECONFIG=/root/codex/kubeconfig_6144665
```

Почему это важно:
- acceptance и load tests зависят от live Kubernetes API
- без правильного `KUBECONFIG` можно получить ложные пустые результаты
- именно этот kubeconfig использовался в финальной validation

Полезно сразу проверить:

```bash
kubectl get nodes
kubectl get --raw=/readyz
```

Если здесь уже есть timeout:
- сначала чинить доступ к кластеру
- только потом запускать acceptance/load tests

## 2. Как устроены тесты в этом репозитории

Тесты разбиты на три уровня:

1. acceptance
2. security
3. load

### Acceptance

Файлы:
- [lifecycle_test.go](../tests/acceptance/lifecycle_test.go)
- [components_test.go](../tests/acceptance/components_test.go)
- [gitops_test.go](../tests/acceptance/gitops_test.go)
- [migration_test.go](../tests/acceptance/migration_test.go)

Что проверяют:
- lifecycle environment manifests
- component behavior
- GitOps self-heal и namespace isolation
- migration CLI

### Security

Файлы:
- [credential_scan_test.go](../tests/security/credential_scan_test.go)
- [netpol_test.go](../tests/security/netpol_test.go)

Что проверяют:
- отсутствие literal credentials в `environments/`
- baseline network policies в live namespaces

### Load

Файл:
- [concurrency_test.go](../tests/load/concurrency_test.go)

Что проверяет:
- 50 environments
- последовательные Git commits
- timing от commit до Argo-processed state
- cleanup и удаление namespaces

## 3. Canonical команды для тестирования

### Acceptance

```bash
go test ./tests/acceptance/... -run TestLifecycle -v
go test ./tests/acceptance/... -run TestComponents -v
go test ./tests/acceptance/... -run TestGitOps -v
go test ./tests/acceptance/... -run TestMigration -v
```

### Security

```bash
go test ./tests/security/... -v
/tmp/kubescape scan platform/
```

### Load

```bash
KUBECONFIG=/root/codex/kubeconfig_6144665 LOAD_TEST_COUNT=50 go test ./tests/load/... -run TestConcurrency -v -timeout 90m
```

## 4. В каком порядке запускать тесты

Если нужно быстро проверить, что система жива:

```bash
go test ./tests/security/... -v
go test ./tests/acceptance/... -run TestLifecycle -v
```

Если менялся provisioning path:

```bash
go test ./tests/acceptance/... -run TestLifecycle -v
go test ./tests/acceptance/... -run TestComponents -v
go test ./tests/acceptance/... -run TestGitOps -v
```

Если менялся CLI:

```bash
go test ./tests/acceptance/... -run TestMigration -v
```

Если менялась capacity / GitOps / reconciliation behavior:

```bash
KUBECONFIG=/root/codex/kubeconfig_6144665 LOAD_TEST_COUNT=50 go test ./tests/load/... -run TestConcurrency -v -timeout 90m
```

## 5. Как мигрировать legacy приложения из docker-compose

Ниже practical workflow.

### Шаг 1. Подготовить compose файл

Исходный compose должен быть доступен локально:

```bash
ls -l ./docker-compose.yml
```

CLI ожидает читаемый compose file. Если файл не найден, migration не начнётся.

### Шаг 2. Прогнать conversion

```bash
./cli/idp migrate --from ./docker-compose.yml --output ./environment.yaml
```

Если binary ещё не собран:

```bash
cd cli
go build -o idp ./cmd/idp
cd ..
./cli/idp migrate --from ./docker-compose.yml --output ./environment.yaml
```

Что делает команда:
- читает compose
- строит `Environment` manifest
- пишет YAML output
- печатает conversion report

Что важно понимать:
- supported service types будут mapped в platform components
- unsupported/unknown services будут сохранены как `enabled: false`
- manual review warnings будут выведены в отчёте

### Шаг 3. Проверить conversion output

Открыть результат:

```bash
sed -n '1,220p' ./environment.yaml
```

Смотреть на:
- `metadata.name`
- `spec.team`
- `spec.owner`
- список `spec.components`
- `enabled: false` у unknown services

Что нормально:
- web-like services -> `type: webapp`
- postgres/db -> `type: postgresql`
- redis -> `type: redis`
- неизвестные сервисы остались в manifest, но disabled

### Шаг 4. Проверить, что исходный compose не был изменён

CLI не должен мутировать исходник.

Практически:

```bash
git diff -- ./docker-compose.yml
```

Если source file под git, diff должен быть пустым.

### Шаг 5. Провалидировать manifest

```bash
./cli/idp validate --file ./environment.yaml
```

Что проверяется:
- schema
- duplicate component names
- replicas bounds
- image tags
- config override keys

Если validation падает:
- сначала исправить manifest локально
- не коммитить его в `environments/`, пока он не валиден

### Шаг 6. Зарегистрировать legacy environment в tracker

Если нужно вести migration lifecycle:

```bash
export IDP_TRACKER_DB=/path/to/tracker.db
./cli/idp register-legacy --name my-legacy-env --team my-team --compose-path ./docker-compose.yml
```

Что это даёт:
- создаётся tracker record
- дальнейшие `complete-migration` / `deprecate-legacy` будут работать по этому env

### Шаг 7. При необходимости экспортировать legacy config

```bash
./cli/idp export --name my-legacy-env --team my-team --output ./legacy-export.yaml
```

Что важно:
- sensitive env vars с `PASSWORD`, `SECRET`, `KEY`, `TOKEN` редактируются в `<REDACTED>`
- export нужен для review/анализа, а не как конечный GitOps manifest

### Шаг 8. Привести output к platform-ready виду

Типичные действия перед commit:
- задать осмысленный `team`
- задать корректный `owner`
- отключить unsupported services через `enabled: false`
- проверить `replicas`
- проверить image tags
- проверить, что секреты не попали literal values в YAML

## 6. Как запускать тестовые environments

Есть два основных способа:

1. напрямую через Git manifest
2. через Backstage scaffolder

### Вариант A. Через Git manifest

#### Шаг 1. Создать manifest

Путь:

```bash
environments/<team>/<env>.yaml
```

Пример:

```yaml
apiVersion: idp.platform.io/v1alpha1
kind: Environment
metadata:
  name: demo
spec:
  owner: group:default/platform
  team: platform
  components:
    - name: web
      type: webapp
      enabled: true
      replicas: 1
```

#### Шаг 2. Commit и push

```bash
git add environments/platform/demo.yaml
git commit -m "Add environment platform/demo"
git push origin feature/phase-1-foundation
```

#### Шаг 3. Проверить generated Argo Application

```bash
kubectl get application -n argocd platform-demo -o yaml
```

Искать:
- `status.sync.status: Synced`
- `status.health.status: Healthy`

#### Шаг 4. Проверить Environment claim

```bash
kubectl get environment -n crossplane-system demo -o yaml
```

Искать:
- `Synced=True`
- `Ready=True`

#### Шаг 5. Проверить namespace

```bash
kubectl get ns env-platform-demo
```

#### Шаг 6. Проверить runtime ресурсы

```bash
kubectl get all -n env-platform-demo
kubectl get configmap,secret,pvc -n env-platform-demo
kubectl get netpol -n env-platform-demo
```

Если всё зелёное:
- test environment поднят успешно

### Вариант B. Через Backstage

Используется template:
- [templates/new-environment/template.yaml](../templates/new-environment/template.yaml)

Фактически flow такой же:
- Backstage делает commit в GitOps branch
- ArgoCD создаёт generated app
- Crossplane создаёт environment resources

После запуска template нужно проверять то же самое:

```bash
kubectl get application -n argocd <team>-<env>
kubectl get environment -n crossplane-system <env>
kubectl get ns env-<team>-<env>
```

## 7. Как обновлять и удалять test environments

### Изменить environment

Редактируешь manifest:

```bash
vim environments/<team>/<env>.yaml
git add environments/<team>/<env>.yaml
git commit -m "Update environment <team>/<env>"
git push origin feature/phase-1-foundation
```

Потом проверяешь:

```bash
kubectl get application -n argocd <team>-<env>
kubectl get environment -n crossplane-system <env>
kubectl get all -n env-<team>-<env>
```

### Отключить конкретный сервис

В manifest:

```yaml
enabled: false
```

Это правильный способ выключить component, не удаляя сам environment.

### Удалить environment полностью

Удаляешь manifest:

```bash
git rm environments/<team>/<env>.yaml
git commit -m "Delete environment <team>/<env>"
git push origin feature/phase-1-foundation
```

Потом проверяешь pruning:

```bash
kubectl get application -n argocd <team>-<env>
kubectl get ns env-<team>-<env>
```

В steady state:
- generated app исчезает
- namespace удаляется

## 8. Как добавлять в test environments новые сервисы

Здесь есть два разных сценария:

1. добавить новый service instance существующего типа
2. добавить новый component type в платформу

### Сценарий A. Добавить новый сервис существующего типа

Это самый частый случай.

Пример: добавить ещё один webapp или database component в конкретный environment.

В manifest:

```yaml
spec:
  components:
    - name: web
      type: webapp
      enabled: true
      replicas: 1
    - name: admin
      type: webapp
      enabled: true
      replicas: 1
```

Что важно:
- `name` должен быть уникален внутри `components[]`
- `type` должен быть уже поддерживаемым
- duplicates будут rejected validation'ом

После commit нужно проверить:

```bash
kubectl get environment -n crossplane-system <env> -o yaml
kubectl get all -n env-<team>-<env>
```

Ожидаем:
- новые resources для `admin`
- существующие ресурсы не ломаются

### Сценарий B. Добавить новый component type в платформу

Это уже platform development, не просто изменение manifest.

Нужно сделать минимум четыре вещи:

1. определить новый тип в converter / migration logic при необходимости
2. добавить новый Crossplane XRD/Composition или другой provisioning mechanism
3. добавить environment-composition mapping этого типа
4. расширить acceptance coverage

Практически путь такой:

#### Шаг 1. Выбрать модель нового типа

Примеры:
- новый stateless app type
- queue / broker
- object storage abstraction
- специализированный worker

Нужно решить:
- это отдельный XRD/composition
- или это разновидность существующего `webapp`

#### Шаг 2. Добавить provisioning definition

Если нужен отдельный type:
- создать новый XRD/composition рядом с existing ones under `platform/crossplane/`
- обновить environment composition, чтобы новый `component.type` мапился на нужный child composite

Ключевая точка:
- [platform/crossplane/environment/composition.yaml](../platform/crossplane/environment/composition.yaml)

#### Шаг 3. Проверить provider/function compatibility

После добавления нового type нужно убедиться, что:
- требуемые providers существуют
- требуемые functions установлены
- generated resources могут быть созданы текущим `kubernetes-provider`

#### Шаг 4. Обновить CLI migration behavior при необходимости

Если новый type должен автоматически обнаруживаться из compose:
- обновить:
  - [cli/pkg/converter/converter.go](../cli/pkg/converter/converter.go)

И потом проверить:

```bash
go test ./tests/acceptance/... -run TestMigration -v
```

#### Шаг 5. Добавить acceptance coverage

Минимум:
- lifecycle для нового type
- provisioning happy path
- cleanup path
- если есть security/network особенности, то и соответствующие tests

## 9. Как понять, какой type использовать в components

Сейчас платформа явно поддерживает:
- `webapp`
- `postgresql`
- `redis`

Выбор:
- HTTP/UI/API/backend process -> `webapp`
- PostgreSQL database -> `postgresql`
- Redis cache -> `redis`

Если сервис не укладывается в эту модель:
- сначала оставь его `enabled: false`
- зафиксируй TODO/manual review
- потом решай, нужен ли новый platform type

## 10. Как работать с unknown services при migration

Когда compose service не мапится:
- converter сохраняет его в manifest как disabled
- добавляет warning/TODO

Это правильное поведение.

Неправильное поведение:
- forcibly mapping unknown service в случайный existing type без понимания runtime semantics

Рекомендуемый workflow:
1. оставить service disabled
2. задокументировать, что именно это за workload
3. решить, можно ли выразить его как `webapp`
4. если нельзя, спроектировать новый component type

## 11. Как понимать результаты acceptance tests

### Lifecycle

Если `TestLifecycle` падает:
- проблема либо в basic GitOps flow
- либо в environment schema/validation
- либо в prune/delete path

### Components

Если `TestComponents` падает:
- проблема в component composition
- db provisioning
- image/replica overrides
- secret hygiene

### GitOps

Если `TestGitOps` падает:
- смотреть ownership model
- generated app drift
- namespace isolation
- pod-to-pod policy behavior

### Migration

Если `TestMigration` падает:
- проблема в CLI command semantics
- conversion/export/validation
- tracker db lifecycle
- report generation

### Load

Если `TestConcurrency` падает:
- сначала проверить branch push path
- потом generated applications
- потом namespace creation
- потом cleanup path

## 12. Практический пример: миграция и запуск простого web приложения

### Исходный compose

```yaml
services:
  web:
    image: nginx:1.27
    ports:
      - "8080:80"
```

### Конвертация

```bash
./cli/idp migrate --from ./docker-compose.yml --output ./environment.yaml
./cli/idp validate --file ./environment.yaml
```

### Review результата

Ожидаемо будет что-то вроде:

```yaml
spec:
  components:
    - name: web
      type: webapp
      enabled: true
      imageTag: "1.27"
```

### Внедрение в GitOps

```bash
mkdir -p environments/platform
cp ./environment.yaml environments/platform/demo-web.yaml
git add environments/platform/demo-web.yaml
git commit -m "Add demo web environment"
git push origin feature/phase-1-foundation
```

### Проверка runtime

```bash
kubectl get application -n argocd platform-demo-web
kubectl get environment -n crossplane-system demo-web
kubectl get ns env-platform-demo-web
kubectl get all -n env-platform-demo-web
```

## 13. Практический пример: добавить Postgres к существующему environment

Допустим, в environment уже есть web component.

Редактируешь:

```yaml
spec:
  components:
    - name: web
      type: webapp
      enabled: true
      replicas: 1
    - name: database
      type: postgresql
      enabled: true
```

Коммитишь:

```bash
git add environments/platform/demo.yaml
git commit -m "Add postgres to platform/demo"
git push origin feature/phase-1-foundation
```

Проверяешь:

```bash
kubectl get all -n env-platform-demo
kubectl get pvc -n env-platform-demo
kubectl get secret -n env-platform-demo
```

Ожидаешь:
- database runtime resources появились
- credentials secret создан
- PVC bound

## 14. Практический пример: добавить новый web-like service

Если у тебя второй HTTP service, часто достаточно ещё одного `webapp` component:

```yaml
spec:
  components:
    - name: web
      type: webapp
      enabled: true
      replicas: 1
    - name: admin
      type: webapp
      enabled: true
      replicas: 1
```

Затем:

```bash
./cli/idp validate --file environments/platform/demo.yaml
git add environments/platform/demo.yaml
git commit -m "Add admin service"
git push origin feature/phase-1-foundation
kubectl get all -n env-platform-demo
```

## 15. Типичные ошибки и как их избегать

### Ошибка: забыли задать `KUBECONFIG`

Симптомы:
- пустые `jsonpath` результаты
- intermittent `NotFound`
- acceptance/load tests ведут себя странно

Решение:

```bash
export KUBECONFIG=/root/codex/kubeconfig_6144665
```

### Ошибка: ищут generated app по неверному имени

Нужно искать:
- `<team>-<env>`

А не:
- raw file path
- namespace name
- component name

### Ошибка: unknown service насильно делают `enabled: true`

Если тип не поддерживается:
- сначала оставить `enabled: false`
- потом решить, нужен ли новый component type

### Ошибка: тестируют platform bug при нестабильном API

Сначала:

```bash
kubectl get nodes
kubectl get --raw=/readyz
```

Только потом запускать тяжёлые tests.

## 16. Recommended Workflow для обычного пользователя платформы

Если цель — просто поднять test environment:

1. написать или сгенерировать `Environment` manifest
2. прогнать `validate`
3. commit/push в GitOps branch
4. дождаться generated `Application`
5. проверить `Environment` claim
6. проверить namespace и workload resources

Если цель — мигрировать legacy compose:

1. `migrate`
2. review output
3. `validate`
4. при необходимости `register-legacy`
5. commit manifest в `environments/`
6. проверить runtime
7. после завершения migration lifecycle:
   - `complete-migration`
   - `deprecate-legacy`

## 17. Summary

Коротко:
- миграция из compose идёт через `idp migrate` + `idp validate`
- запуск test environment идёт через Git commit или Backstage template
- добавление новых сервисов существующих типов делается прямо в `spec.components`
- добавление новых component types требует platform development в Crossplane/CLI/tests
- основной operational успех определяется по цепочке:
  - Git manifest
  - generated Argo `Application`
  - `Environment` claim
  - namespace
  - runtime resources
