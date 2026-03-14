# Adding A New Platform Component Type

## Назначение

Этот документ — отдельный подробный guide по добавлению нового platform
component type.

Он нужен в ситуации, когда существующих типов:
- `webapp`
- `postgresql`
- `redis`

уже недостаточно, и платформе нужен новый поддерживаемый `spec.components[].type`.

Документ описывает:
- как принять решение, нужен ли новый type вообще
- какие файлы менять
- в каком порядке безопаснее это делать
- как проверять каждый шаг
- как обновить CLI migration и acceptance coverage

## See Also

- [TESTING_GUIDE.md](TESTING_GUIDE.md)
- [USER_JOURNEYS.md](USER_JOURNEYS.md)
- [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)
- [GLOSSARY.md](GLOSSARY.md)
- [status.md](status.md)

## 1. Когда действительно нужен новый type

Новый type нужен не всегда.

Сначала ответь на вопрос:
- это действительно новая runtime-модель
- или это просто ещё один вариант уже существующего `webapp`

### Обычно НЕ нужен новый type

Если workload:
- это HTTP/API/backend service
- ему нужен `Deployment` + `Service` + `ConfigMap`
- он масштабируется как обычное stateless приложение

то чаще всего достаточно `webapp`.

### Обычно нужен новый type

Если workload:
- не должен публиковаться как обычный service
- имеет другую lifecycle-модель
- требует другой набор Kubernetes resources
- требует другого default image/config contract
- требует другой security/runtime policy

Примеры:
- `worker`
- `cronjob`
- `queue`
- `broker`
- `objectstore`

## 2. Что значит “добавить новый type” в этой платформе

В этой платформе новый component type почти всегда означает все четыре слоя:

1. provisioning layer
2. environment composition mapping
3. CLI conversion/migration layer
4. acceptance/testing layer

То есть недостаточно просто разрешить строку `type: worker` в YAML.

Нужно, чтобы платформа реально умела:
- принять этот type в `Environment`
- сгенерировать child composite или другой provisioning path
- создать runtime resources
- удалить их обратно
- при необходимости смапить его из legacy `docker-compose`

## 3. Рекомендуемый порядок работы

Безопасный порядок такой:

1. определить target behavior нового типа
2. создать новый XRD
3. создать новый Composition
4. встроить mapping в `Environment` Composition
5. проверить render/validate
6. подключить type в CLI converter, если это нужно
7. добавить acceptance tests
8. прогнать live smoke

Это лучше, чем начинать с CLI или с миграции, потому что сначала должен
существовать реальный provisioning path.

## 4. Пример: добавляем тип `worker`

Ниже примерный путь на условном type `worker`.

Смысл примера:
- `worker` похож на `webapp`
- но не требует Service
- работает как фоновый process
- может иметь replicas

Это только пример структуры работы. Финальные runtime details нужно подгонять
под конкретный сервис.

## 5. Шаг 1. Зафиксировать контракт нового type

Сначала опиши на бумаге или в PR description:

- какой Kubernetes shape должен получиться
- какие поля нужны пользователю в `spec.components[]`
- какие defaults нужны
- как работает `enabled: false`
- нужны ли volume/secret/db integration

### Минимальный контракт для `worker`

Пример:

```yaml
spec:
  components:
    - name: jobs
      type: worker
      enabled: true
      replicas: 2
      imageTag: 1.0.0
      configOverrides:
        QUEUE_NAME: emails
```

Ожидаемое runtime behavior:
- создаётся `Deployment`
- `Service` не создаётся
- `replicas` поддерживается
- `enabled: false` удаляет runtime resources

## 6. Шаг 2. Создать новый XRD

Сделай новую директорию:

```text
platform/crossplane/worker/
```

И новый файл:

```text
platform/crossplane/worker/xrd.yaml
```

Ориентируйся на существующие:
- [platform/crossplane/webapp/xrd.yaml](../platform/crossplane/webapp/xrd.yaml)
- [platform/crossplane/postgresql/xrd.yaml](../platform/crossplane/postgresql/xrd.yaml)
- [platform/crossplane/redis/xrd.yaml](../platform/crossplane/redis/xrd.yaml)

### Что должно быть в XRD

Минимально:
- новый composite kind
- schema для `spec.parameters`
- поля, которые нужны composition

Примерная идея:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: CompositeResourceDefinition
metadata:
  name: xworkerinstances.idp.platform.io
spec:
  group: idp.platform.io
  names:
    kind: XWorkerInstance
    plural: xworkerinstances
  claimNames:
    kind: WorkerInstance
    plural: workerinstances
  versions:
    - name: v1alpha1
      served: true
      referenceable: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                parameters:
                  type: object
                  properties:
                    environmentName:
                      type: string
                    team:
                      type: string
                    componentName:
                      type: string
                    namespace:
                      type: string
                    imageRepository:
                      type: string
                    imageTag:
                      type: string
                    replicas:
                      type: integer
                    configOverrides:
                      type: object
                      additionalProperties:
                        type: string
```

### Практический совет

Не начинай с “идеального” schema.

Сначала включи только реально используемые поля:
- `namespace`
- `componentName`
- `imageRepository`
- `imageTag`
- `replicas`
- `configOverrides`

Лишние поля только усложнят render/debug.

## 7. Шаг 3. Создать новый Composition

Файл:

```text
platform/crossplane/worker/composition.yaml
```

Ориентируйся на:
- [platform/crossplane/webapp/composition.yaml](../platform/crossplane/webapp/composition.yaml)

### Что должен делать Composition

Для `worker` в минимальном варианте:
- создать `ConfigMap`
- создать `Deployment`
- при необходимости создать `Secret`

### Упрощённый пример shape

Ниже не production-ready манифест, а ориентир по структуре:

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: worker-composition
spec:
  compositeTypeRef:
    apiVersion: idp.platform.io/v1alpha1
    kind: XWorkerInstance
  mode: Pipeline
  pipeline:
    - step: render
      functionRef:
        name: function-go-templating
      input:
        apiVersion: gotemplating.fn.crossplane.io/v1beta1
        kind: GoTemplate
        source: Inline
        inline:
          template: |
            apiVersion: kubernetes.crossplane.io/v1alpha2
            kind: Object
            metadata:
              name: {{ .observed.composite.resource.spec.parameters.componentName }}-config
            spec:
              providerConfigRef:
                name: kubernetes-provider
              forProvider:
                manifest:
                  apiVersion: v1
                  kind: ConfigMap
                  metadata:
                    name: {{ .observed.composite.resource.spec.parameters.componentName }}-config
                    namespace: {{ .observed.composite.resource.spec.parameters.namespace }}
                  data:
                    WORKER_MODE: default
            ---
            apiVersion: kubernetes.crossplane.io/v1alpha2
            kind: Object
            metadata:
              name: {{ .observed.composite.resource.spec.parameters.componentName }}-deployment
            spec:
              providerConfigRef:
                name: kubernetes-provider
              forProvider:
                manifest:
                  apiVersion: apps/v1
                  kind: Deployment
                  metadata:
                    name: {{ .observed.composite.resource.spec.parameters.componentName }}
                    namespace: {{ .observed.composite.resource.spec.parameters.namespace }}
                  spec:
                    replicas: {{ .observed.composite.resource.spec.parameters.replicas }}
                    selector:
                      matchLabels:
                        app.kubernetes.io/instance: {{ .observed.composite.resource.spec.parameters.componentName }}
                    template:
                      metadata:
                        labels:
                          app.kubernetes.io/instance: {{ .observed.composite.resource.spec.parameters.componentName }}
                      spec:
                        containers:
                          - name: worker
                            image: {{ .observed.composite.resource.spec.parameters.imageRepository }}:{{ .observed.composite.resource.spec.parameters.imageTag }}
    - step: ready
      functionRef:
        name: function-auto-ready
```

### Что проверить сразу

Проверь, что composition:
- использует существующий `kubernetes-provider`
- использует Functions, уже установленные в платформе
- не требует дополнительных provider'ов без необходимости

## 8. Шаг 4. Подключить type в `Environment` Composition

Ключевой файл:
- [platform/crossplane/environment/composition.yaml](../platform/crossplane/environment/composition.yaml)

Именно здесь `spec.components[]` мапится в child composites.

### Что нужно сделать

Добавить новую ветку логики:
- если `component.type == "worker"`
- создать `XWorkerInstance`

### Идея ожидаемого child composite

Пример того, что должен получить render:

```yaml
apiVersion: idp.platform.io/v1alpha1
kind: XWorkerInstance
metadata:
  name: demo-jobs
spec:
  parameters:
    environmentName: demo
    team: platform
    componentName: jobs
    namespace: env-platform-demo
    imageRepository: docker.io/example/worker
    imageTag: 1.0.0
    replicas: 2
    configOverrides:
      QUEUE_NAME: emails
```

### Что важно

Нужно сохранить текущие свойства платформы:
- disabled component не попадает в desired state
- остальные types не ломаются
- namespace derivation остаётся той же

## 9. Шаг 5. Подключить новый XRD/Composition в GitOps path

Если ты добавил:
- `platform/crossplane/worker/xrd.yaml`
- `platform/crossplane/worker/composition.yaml`

то они должны попасть в тот repo-hosted path, который уже синхронизируется
через ArgoCD Crossplane app.

Поскольку [platform/argocd/app-crossplane.yaml](../platform/argocd/app-crossplane.yaml)
уже тянет `platform/crossplane/`, обычно отдельной wiring-работы не нужно.

Но проверить нужно обязательно.

### Проверка

```bash
kubectl apply --dry-run=client -f platform/crossplane/
kubectl get xrd
kubectl get composition
```

Ожидаешь увидеть новый:
- `xworkerinstances.idp.platform.io`
- `worker-composition`

## 10. Шаг 6. Добавить converter mapping, если нужен compose migration

Если новый type должен автоматически обнаруживаться из `docker-compose`,
меняй:
- [cli/pkg/converter/converter.go](../cli/pkg/converter/converter.go)

### Что обычно нужно сделать

Найти в converter место, где сервисы мапятся в platform types, и добавить:
- новые heuristics
- новый output type
- warnings/report entries, если mapping частичный

### Пример логики

Например, если service name или command указывает на worker:

```go
if strings.Contains(service.Name, "worker") || strings.Contains(command, "celery") {
    component.Type = "worker"
}
```

### Важное правило

Не делай агрессивный auto-detection без explainability.

Если converter не уверен:
- лучше оставить `enabled: false`
- добавить warning/TODO в report

чем silently mis-map сервис в неверный type.

## 11. Шаг 7. Добавить validator logic, если у нового type есть свои правила

Если `worker` использует только общие поля:
- возможно, отдельных validator changes не нужно

Если у нового type появляются свои специальные поля:
- тогда нужно обновить validator contract и schema checks

Смотри:
- [cli/pkg/validator/validator.go](../cli/pkg/validator/validator.go)

Примеры, когда нужны validator rules:
- `schedule` для cron-like type
- `queueName` обязателен
- type-specific config keys ограничены whitelist'ом

## 12. Шаг 8. Добавить acceptance и unit coverage

Минимальный набор тестов для нового type:

1. render/validation tests для XRD + Composition
2. environment-composition mapping test
3. acceptance happy path
4. cleanup/disable path
5. migration test, если type участвует в converter

### Где смотреть существующие тесты

- acceptance:
  - [../tests/acceptance/lifecycle_test.go](../tests/acceptance/lifecycle_test.go)
  - [../tests/acceptance/components_test.go](../tests/acceptance/components_test.go)
- migration:
  - [../tests/acceptance/migration_test.go](../tests/acceptance/migration_test.go)
- converter tests:
  - [../cli/pkg/converter/converter_test.go](../cli/pkg/converter/converter_test.go)

### Минимальный acceptance сценарий

Нужен хотя бы такой live test:
- commit environment с `type: worker`
- дождаться generated Argo Application
- дождаться `Environment Ready=True`
- проверить namespace
- проверить `Deployment/jobs`
- потом отключить `enabled: false`
- проверить, что runtime resources удалены

## 13. Шаг 9. Прогнать локальную и live validation

Минимальный список команд:

```bash
kubectl apply --dry-run=client -f platform/crossplane/
go test ./cli/... ./tests/acceptance/... ./tests/security/... -v
```

И отдельно:

```bash
go test ./tests/acceptance/... -run TestComponents -v
go test ./tests/acceptance/... -run TestLifecycle -v
go test ./tests/acceptance/... -run TestMigration -v
```

Если ты добавил новый type только в provisioning, а migration не менял:
- `TestMigration` может не требоваться

Но если type должен участвовать в compose conversion:
- его обязательно запускать

## 14. Шаг 10. Сделать smoke manifest

Полезно создать временный manifest:

```yaml
apiVersion: idp.platform.io/v1alpha1
kind: Environment
metadata:
  name: worker-smoke
spec:
  owner: group:default/platform
  team: platform
  components:
    - name: jobs
      type: worker
      enabled: true
      replicas: 1
      imageTag: 1.0.0
```

Потом:

```bash
git add environments/platform/worker-smoke.yaml
git commit -m "Add worker smoke environment"
git push origin feature/phase-1-foundation
```

Проверки:

```bash
kubectl get application -n argocd platform-worker-smoke -o yaml
kubectl get environment -n crossplane-system worker-smoke -o yaml
kubectl get ns env-platform-worker-smoke
kubectl get all -n env-platform-worker-smoke
```

## 15. Типичные ошибки

### Ошибка 1. Добавили type в converter, но не в provisioning

Симптом:
- migration выдаёт `type: worker`
- live provisioning не создаёт resources

Причина:
- отсутствует mapping в `platform/crossplane/environment/composition.yaml`

### Ошибка 2. Добавили новый child composite, но не создали XRD

Симптом:
- Crossplane composition errors
- unknown kind / cannot render resource

Причина:
- missing `xrd.yaml`

### Ошибка 3. Создали XRD/Composition, но они невалидны для live Crossplane schema

Симптом:
- ArgoCD sync failures
- XRD rejected server-side

Причина:
- schema drift между примером и live Crossplane API

Решение:
- всегда проверять live `kubectl apply --dry-run=client`
- и live cluster acceptance, а не только offline render

### Ошибка 4. Новый type требует extra provider/function, но они не установлены

Симптом:
- composite created, but reconcile stuck
- FunctionRevision / provider runtime errors

Причина:
- platform runtime prerequisites не добавлены в `platform/crossplane/`

### Ошибка 5. Acceptance test проверяет не тот ownership layer

Симптом:
- ожидали, что Argo восстановит `Deployment`
- а Argo управляет только `Environment` claim

Решение:
- проверять drift там, где ownership реально у Argo

## 16. Чеклист перед merge

Перед merge ответь “да” на всё:

- новый XRD добавлен
- новый Composition добавлен
- `Environment` Composition умеет мапить новый type
- `enabled: false` работает
- runtime resources создаются
- runtime resources удаляются
- validator обновлён, если нужно
- converter обновлён, если нужно
- tests добавлены
- live smoke прошёл
- docs обновлены

## 17. Какие docs обновить после добавления нового type

Минимум проверь:
- [TESTING_GUIDE.md](TESTING_GUIDE.md)
- [USER_JOURNEYS.md](USER_JOURNEYS.md)
- [GLOSSARY.md](GLOSSARY.md)
- [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)
- при необходимости runbooks в `docs/runbooks/`

Если новый type должен участвовать в migration:
- обнови migration-related guidance тоже

## 18. Краткий practical summary

Если максимально коротко, путь такой:

1. создай `platform/crossplane/<type>/xrd.yaml`
2. создай `platform/crossplane/<type>/composition.yaml`
3. добавь mapping в `platform/crossplane/environment/composition.yaml`
4. проверь `kubectl apply --dry-run=client -f platform/crossplane/`
5. если нужен compose migration, обнови `cli/pkg/converter/converter.go`
6. добавь tests
7. прогоняй live smoke environment

Если пропустить любой из этих шагов, новый type почти наверняка будет “частично добавлен”, но не реально usable.
