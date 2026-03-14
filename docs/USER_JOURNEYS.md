# User Journeys

## Назначение

Этот документ описывает типовые end-to-end сценарии использования платформы.

Он нужен, когда хочется не читать абстрактную архитектуру, а пройти по понятному пути:
- что делает пользователь
- что делает платформа
- что проверить на каждом шаге
- где искать проблему, если результат не появился

## See Also

- [INDEX.md](../INDEX.md)
- [README.md](../README.md)
- [TESTING_GUIDE.md](TESTING_GUIDE.md)
- [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)
- [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)

## Journey 1. Из `docker-compose` в живой test environment

### Цель

Инженер берёт legacy `docker-compose.yml`, превращает его в platform manifest и поднимает новый test environment.

### Шаг 1. Подготовить compose

У пользователя есть:

```bash
./docker-compose.yml
```

### Шаг 2. Запустить migration

```bash
./cli/idp migrate --from ./docker-compose.yml --output ./environment.yaml
./cli/idp validate --file ./environment.yaml
```

### Что ожидается

- создаётся `environment.yaml`
- CLI печатает conversion report
- validation проходит без ошибок

### Что делать, если не получилось

Если `validate` падает:
- открыть output manifest
- проверить duplicate names
- проверить `replicas`
- посмотреть warnings/TODO по unsupported services

### Шаг 3. Привести manifest к финальному виду

Проверить:
- `spec.team`
- `spec.owner`
- `enabled: false` у неизвестных сервисов
- image tags
- replicas

### Шаг 4. Положить manifest в GitOps repo

```bash
mkdir -p environments/platform
cp ./environment.yaml environments/platform/demo.yaml
git add environments/platform/demo.yaml
git commit -m "Add demo environment"
git push origin feature/phase-1-foundation
```

### Шаг 5. Проверить Argo / Crossplane / runtime

```bash
export KUBECONFIG=/root/codex/kubeconfig_6144665
kubectl get application -n argocd platform-demo
kubectl get environment -n crossplane-system demo
kubectl get ns env-platform-demo
kubectl get all -n env-platform-demo
```

### Успешный результат

- generated `Application` есть и синхронизирован
- `Environment` claim есть и готов
- namespace создан
- workload resources присутствуют

## Journey 2. Создать test environment через Backstage

### Цель

Пользователь не редактирует Git вручную, а запускает новый environment через портал.

### Шаг 1. Открыть Backstage

Пользователь логинится через OIDC.

### Шаг 2. Открыть scaffolder template

Используется template:
- [template.yaml](../templates/new-environment/template.yaml)

### Шаг 3. Заполнить параметры

Обычно указываются:
- environment name
- team
- owner
- включённые компоненты

### Шаг 4. Запустить scaffolder task

Backstage делает:
- render skeleton
- commit в GitOps branch
- push generated manifest

### Шаг 5. Проверить результат

```bash
kubectl get application -n argocd <team>-<env>
kubectl get environment -n crossplane-system <env>
kubectl get ns env-<team>-<env>
```

### Успешный результат

- environment появляется без ручного Git-edit
- generated app создаётся автоматически
- namespace и workload resources появляются

### Если проблема

Проверять по цепочке:
1. Backstage task logs
2. Git commit на branch
3. generated Argo `Application`
4. `Environment` claim
5. namespace/runtime resources

## Journey 3. Добавить новый сервис в уже существующий environment

### Цель

В существующий environment нужно добавить ещё один service instance поддерживаемого типа.

### Пример

Есть environment с `web`, нужно добавить `admin`.

### Шаг 1. Изменить manifest

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

### Шаг 2. Провалидировать

```bash
./cli/idp validate --file environments/platform/demo.yaml
```

### Шаг 3. Commit и push

```bash
git add environments/platform/demo.yaml
git commit -m "Add admin service to demo"
git push origin feature/phase-1-foundation
```

### Шаг 4. Проверить runtime

```bash
kubectl get environment -n crossplane-system demo
kubectl get all -n env-platform-demo
```

### Успешный результат

- новые runtime resources для `admin` появились
- существующие ресурсы остались рабочими

### Если проблема

Смотреть:
- duplicate component names
- правильность `type`
- generated app status
- composition/runtime events

## Journey 4. Отключить сервис, не удаляя environment

### Цель

Нужно временно убрать один из компонентов из environment, но сам environment оставить.

### Шаг 1. Перевести component в disabled

```yaml
enabled: false
```

### Шаг 2. Commit и push

```bash
git add environments/<team>/<env>.yaml
git commit -m "Disable component in <team>/<env>"
git push origin feature/phase-1-foundation
```

### Шаг 3. Проверить удаление runtime resources

```bash
kubectl get all -n env-<team>-<env>
```

### Успешный результат

- только disabled component исчезает
- manifest продолжает существовать
- остальные components работают

## Journey 5. Полностью удалить environment

### Цель

Нужно убрать весь test environment из платформы.

### Шаг 1. Удалить manifest

```bash
git rm environments/<team>/<env>.yaml
git commit -m "Delete environment <team>/<env>"
git push origin feature/phase-1-foundation
```

### Шаг 2. Проверить pruning

```bash
kubectl get application -n argocd <team>-<env>
kubectl get ns env-<team>-<env>
```

### Успешный результат

- generated app удалён
- namespace удалён
- runtime resources исчезли

## Journey 6. Зарегистрировать legacy environment и завершить migration lifecycle

### Цель

Команда хочет не просто мигрировать manifest, а ещё и отслеживать lifecycle legacy environment.

### Шаг 1. Зарегистрировать legacy environment

```bash
export IDP_TRACKER_DB=/path/to/tracker.db
./cli/idp register-legacy --name shop --team alpha --compose-path ./docker-compose.yml
```

### Шаг 2. Выполнить migration в GitOps path

```bash
./cli/idp migrate --from ./docker-compose.yml --output ./environment.yaml
./cli/idp validate --file ./environment.yaml
```

Потом:
- commit manifest
- дождаться, что environment поднялся

### Шаг 3. Отметить migration completed

Когда migration реально завершена:

```bash
./cli/idp complete-migration --env-id <id>
```

Важно:
- команда сработает только если запись уже в `validated`

### Шаг 4. Проверить deprecation gate

```bash
IDP_REPORT_DIR=./reports ./cli/idp deprecate-legacy --dry-run
```

Если все legacy environments completed:

```bash
IDP_REPORT_DIR=./reports ./cli/idp deprecate-legacy --confirm
```

### Успешный результат

- deprecation gate passes
- создаётся `deprecation-report-<date>.md`
- report содержит environment IDs и timestamps

## Journey 7. Добавить новый platform component type

### Цель

Команде нужен новый тип сервиса, которого сейчас нет среди:
- `webapp`
- `postgresql`
- `redis`

### Это уже не пользовательский, а платформенный сценарий

Нужно:
1. спроектировать новый type
2. добавить provisioning layer
3. при необходимости обновить CLI converter
4. добавить tests

### Практический путь

#### Шаг 1. Решить, действительно ли нужен новый type

Вопрос:
- это реально отдельная runtime-модель
- или существующий `webapp` уже достаточен

#### Шаг 2. Добавить Crossplane definition

Понадобится:
- новый XRD/composition или эквивалентный provisioning path
- изменение [platform/crossplane/environment/composition.yaml](../platform/crossplane/environment/composition.yaml)

#### Шаг 3. Добавить mapping в CLI converter при необходимости

Файл:
- [cli/pkg/converter/converter.go](../cli/pkg/converter/converter.go)

#### Шаг 4. Добавить acceptance coverage

Минимум:
- create path
- reconcile path
- cleanup path

### Когда считать работу завершённой

Когда новый type:
- можно описать в `Environment` manifest
- поднимается через GitOps
- проходит validation
- покрыт tests

## Journey 8. Диагностика: manifest в Git есть, но environment не появляется

### Признак

Команда уже запушила manifest, но namespace или resources не появились.

### Правильный порядок проверки

1. Git branch

```bash
git ls-remote --heads origin feature/phase-1-foundation
```

2. Generated app

```bash
kubectl get application -n argocd <team>-<env> -o yaml
```

3. Claim

```bash
kubectl get environment -n crossplane-system <env> -o yaml
```

4. Namespace

```bash
kubectl get ns env-<team>-<env>
```

5. Runtime resources

```bash
kubectl get all -n env-<team>-<env>
```

### Интерпретация

- файла нет в remote Git -> проблема до Argo
- app нет -> проблема в ApplicationSet/Argo reconcile
- app `Synced`, claim нет -> проблема apply layer
- claim есть, namespace нет -> проблема Crossplane composition/provider
- namespace есть, pods broken -> проблема runtime/workload config

## Journey 9. Быстрый demo path для новой команды

Если нужно показать платформу за 10 минут:

1. взять простой `docker-compose.yml`
2. сделать:

```bash
./cli/idp migrate --from ./docker-compose.yml --output ./environment.yaml
./cli/idp validate --file ./environment.yaml
```

3. положить manifest в `environments/<team>/`
4. commit/push
5. показать:

```bash
kubectl get application -n argocd <team>-<env>
kubectl get environment -n crossplane-system <env>
kubectl get ns env-<team>-<env>
kubectl get all -n env-<team>-<env>
```

Это лучший короткий end-to-end walkthrough.

## Итог

Если кратко:
- обычный пользователь платформы чаще всего идёт по Journey 1, 2, 3 или 4
- migration/legacy lifecycle покрывают Journey 6
- platform engineering для новых типов — это Journey 7
- для incident/debug самый полезный сценарий — Journey 8
