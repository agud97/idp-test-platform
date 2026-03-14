# Adding 10 New Webapps To The Platform

## Назначение

Этот документ описывает практический сценарий:

- нужно добавить не новый component type
- а сразу много новых приложений
- все они подходят под существующий type `webapp`
- у них разные:
  - имена
  - container images
  - image tags
  - environment variables
  - replica counts

Это типовой сценарий для команды, которая хочет быстро завести в платформу
несколько сервисов одного семейства без изменения platform architecture.

## See Also

- [TESTING_GUIDE.md](TESTING_GUIDE.md)
- [USER_JOURNEYS.md](USER_JOURNEYS.md)
- [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)
- [ADDING_NEW_PLATFORM_COMPONENT_TYPE.md](ADDING_NEW_PLATFORM_COMPONENT_TYPE.md)
- [examples/ten-webapps-environment.yaml](examples/ten-webapps-environment.yaml)
- [status.md](status.md)

## 1. Когда этот guide подходит

Используй этот путь, если все новые приложения:
- запускаются как обычные stateless workloads
- должны работать как `Deployment`
- могут быть описаны через существующий `type: webapp`
- не требуют нового provisioning mechanism

Примеры:
- `api`
- `admin`
- `frontend`
- `billing`
- `notifications`
- `gateway`
- `search`
- `worker-ui`
- `backoffice`
- `reporting-api`

## 2. Что именно считается “добавить приложение”

В этой платформе “добавить приложение” обычно означает:

1. выбрать environment manifest
2. добавить новый entry в `spec.components[]`
3. задать для него:
   - `name`
   - `type: webapp`
   - `enabled`
   - `imageRepository`
   - `imageTag`
   - `replicas`
   - `configOverrides` при необходимости
4. закоммитить manifest в Git
5. дождаться ArgoCD + Crossplane reconcile

Важно:
- новый XRD не нужен
- новый Composition не нужен
- менять `platform/crossplane/webapp/` не нужно, если контракт уже покрывает твой случай

## 3. Что подготовить до редактирования YAML

Перед тем как писать manifest, собери таблицу по всем 10 приложениям.

Рекомендуемый шаблон:

| app_name | image_repository | image_tag | replicas | enabled | env_vars | notes |
|---|---|---:|---:|---|---|---|
| api | ghcr.io/acme/api | 1.4.2 | 2 | true | PORT=8080, LOG_LEVEL=info | main backend |
| admin | ghcr.io/acme/admin | 2.1.0 | 1 | true | PORT=8080 | admin UI |
| frontend | ghcr.io/acme/frontend | 2026.03.15 | 2 | true | PORT=8080, API_URL=http://api | public UI |
| billing | ghcr.io/acme/billing | 0.9.3 | 1 | true | PORT=8080 | billing API |
| notifications | ghcr.io/acme/notifications | 1.2.0 | 1 | true | PORT=8080 | notifications API |
| gateway | ghcr.io/acme/gateway | 3.0.1 | 2 | true | PORT=8080 | edge gateway |
| search | ghcr.io/acme/search | 1.7.4 | 1 | true | PORT=8080 | search API |
| worker-ui | ghcr.io/acme/worker-ui | 0.5.0 | 1 | false | PORT=8080 | not ready yet |
| backoffice | ghcr.io/acme/backoffice | 4.2.1 | 1 | true | PORT=8080 | internal UI |
| reporting-api | ghcr.io/acme/reporting-api | 2.0.0 | 1 | true | PORT=8080 | reports |

### Почему это важно

Если сначала не собрать таблицу:
- в YAML легко ошибиться в имени
- можно перепутать image/tag между сервисами
- можно продублировать component name
- можно забыть, какие env vars вообще нужны

## 4. Куда именно добавлять эти 10 приложений

Есть два варианта:

1. в один существующий environment
2. в новый environment

### Вариант A. В существующий environment

Если у тебя уже есть файл:

```text
environments/<team>/<env>.yaml
```

то ты просто расширяешь его `spec.components[]`.

### Вариант B. В новый environment

Если environment ещё нет:
- создаёшь новый файл в `environments/<team>/<env>.yaml`
- сразу описываешь все 10 приложений внутри него

## 5. Базовый шаблон manifest для 10 webapp-компонентов

Ниже практический пример одного environment с 10 приложениями.

Готовый отдельный пример в репозитории:
- [examples/ten-webapps-environment.yaml](examples/ten-webapps-environment.yaml)

```yaml
apiVersion: idp.platform.io/v1alpha1
kind: Environment
metadata:
  name: retail-suite
spec:
  owner: group:default/platform
  team: retail
  components:
    - name: api
      type: webapp
      enabled: true
      imageRepository: ghcr.io/acme/api
      imageTag: 1.4.2
      replicas: 2
      configOverrides:
        PORT: "8080"
        LOG_LEVEL: info

    - name: admin
      type: webapp
      enabled: true
      imageRepository: ghcr.io/acme/admin
      imageTag: 2.1.0
      replicas: 1
      configOverrides:
        PORT: "8080"

    - name: frontend
      type: webapp
      enabled: true
      imageRepository: ghcr.io/acme/frontend
      imageTag: "2026.03.15"
      replicas: 2
      configOverrides:
        PORT: "8080"
        API_URL: http://api

    - name: billing
      type: webapp
      enabled: true
      imageRepository: ghcr.io/acme/billing
      imageTag: 0.9.3
      replicas: 1
      configOverrides:
        PORT: "8080"

    - name: notifications
      type: webapp
      enabled: true
      imageRepository: ghcr.io/acme/notifications
      imageTag: 1.2.0
      replicas: 1
      configOverrides:
        PORT: "8080"

    - name: gateway
      type: webapp
      enabled: true
      imageRepository: ghcr.io/acme/gateway
      imageTag: 3.0.1
      replicas: 2
      configOverrides:
        PORT: "8080"

    - name: search
      type: webapp
      enabled: true
      imageRepository: ghcr.io/acme/search
      imageTag: 1.7.4
      replicas: 1
      configOverrides:
        PORT: "8080"

    - name: worker-ui
      type: webapp
      enabled: false
      imageRepository: ghcr.io/acme/worker-ui
      imageTag: 0.5.0
      replicas: 1
      configOverrides:
        PORT: "8080"

    - name: backoffice
      type: webapp
      enabled: true
      imageRepository: ghcr.io/acme/backoffice
      imageTag: 4.2.1
      replicas: 1
      configOverrides:
        PORT: "8080"

    - name: reporting-api
      type: webapp
      enabled: true
      imageRepository: ghcr.io/acme/reporting-api
      imageTag: 2.0.0
      replicas: 1
      configOverrides:
        PORT: "8080"
```

## 6. Как выбрать правильные поля для каждого приложения

### `name`

Должно быть:
- уникальным внутри `components[]`
- коротким
- понятным

Хорошо:
- `api`
- `admin`
- `gateway`
- `reporting-api`

Плохо:
- `service`
- `app`
- `backend1`

### `type`

Для твоего кейса всегда:

```yaml
type: webapp
```

### `enabled`

Используй:
- `true` для готовых к запуску приложений
- `false` для приложений, которые пока нужно оставить в manifest, но не поднимать

Это удобно, если из 10 приложений 8 готовы, а 2 ещё нет.

### `imageRepository`

Задавай без tag.

Хорошо:

```yaml
imageRepository: ghcr.io/acme/api
```

Плохо:

```yaml
imageRepository: ghcr.io/acme/api:1.4.2
```

Tag должен жить отдельно в `imageTag`.

### `imageTag`

Храни отдельно:

```yaml
imageTag: 1.4.2
```

или:

```yaml
imageTag: "2026.03.15"
```

Если tag похож на число или дату, лучше брать в кавычки.

### `replicas`

Используй:
- `1` для большинства внутренних UI/API
- `2` или `3` для более важных сервисов
- `0` только если нужно оставить resources, но без running pods

Ограничение платформы:
- `0–50`

### `configOverrides`

Это место для несекретных environment-style параметров.

Подходит для:
- `PORT`
- `LOG_LEVEL`
- `FEATURE_FLAG`
- `API_URL`
- `PUBLIC_BASE_URL`

Не подходит для:
- паролей
- токенов
- приватных ключей

## 7. Как добавлять environment variables правильно

### Что можно класть в `configOverrides`

Безопасные примеры:

```yaml
configOverrides:
  PORT: "8080"
  LOG_LEVEL: info
  FEATURE_X_ENABLED: "true"
  API_URL: http://api
```

### Что нельзя класть

Нельзя коммитить literal secrets:

```yaml
configOverrides:
  DB_PASSWORD: supersecret
  JWT_SECRET: abc123
```

Это нарушит security expectations платформы.

### Практическое правило

Если значение:
- секретное
- выдаётся только runtime
- не должно храниться в Git

то его нельзя помещать в `configOverrides`.

## 8. Как лучше вводить 10 приложений: сразу или поэтапно

Есть два подхода.

### Подход A. Все 10 сразу

Подходит, если:
- все сервисы уже готовы
- образы гарантированно pullable
- env vars уже известны
- ты хочешь один общий environment

Плюсы:
- один commit
- один reconcile cycle
- целостный snapshot среды

Минусы:
- сложнее debug, если ломается один из десяти

### Подход B. Волнами

Рекомендуемый практический путь:

1. сначала 2 приложения
2. потом 3–4
3. потом оставшиеся

Плюсы:
- проще найти проблему
- меньше blast radius
- быстрее понять, какой конкретно app ломает rollout

Минусы:
- больше commits

### Моя рекомендация

Для 10 новых приложений лучше идти волнами:
- wave 1: `api`, `admin`
- wave 2: `frontend`, `gateway`, `search`
- wave 3: `billing`, `notifications`, `backoffice`, `reporting-api`
- `worker-ui` можно оставить `enabled: false` до готовности

## 9. Подшаговый workflow

### Шаг 1. Создай рабочую таблицу

Собери по всем 10 приложениям:
- name
- imageRepository
- imageTag
- replicas
- env vars
- enabled

### Шаг 2. Создай или открой environment manifest

Например:

```bash
vim environments/retail/retail-suite.yaml
```

### Шаг 3. Внеси первые 2 приложения

Не начинай сразу со всех 10, если нет 100% уверенности.

Пример первой волны:

```yaml
components:
  - name: api
    type: webapp
    enabled: true
    imageRepository: ghcr.io/acme/api
    imageTag: 1.4.2
    replicas: 2
    configOverrides:
      PORT: "8080"

  - name: admin
    type: webapp
    enabled: true
    imageRepository: ghcr.io/acme/admin
    imageTag: 2.1.0
    replicas: 1
    configOverrides:
      PORT: "8080"
```

### Шаг 4. Провалидируй manifest локально

```bash
./cli/idp validate --file environments/retail/retail-suite.yaml
```

Если binary ещё не собран:

```bash
cd cli
go build -o idp ./cmd/idp
cd ..
./cli/idp validate --file environments/retail/retail-suite.yaml
```

### Шаг 5. Закоммить и запушь

```bash
git add environments/retail/retail-suite.yaml
git commit -m "Add wave 1 webapps to retail-suite"
git push origin feature/phase-1-foundation
```

### Шаг 6. Проверь generated Argo Application

```bash
export KUBECONFIG=/root/codex/kubeconfig_6144665
kubectl get application -n argocd retail-retail-suite -o yaml
```

Ищи:
- `status.sync.status: Synced`
- `status.health.status: Healthy`

### Шаг 7. Проверь Environment claim

```bash
kubectl get environment -n crossplane-system retail-suite -o yaml
```

Ищи:
- `Synced=True`
- `Ready=True`

### Шаг 8. Проверь namespace

```bash
kubectl get ns env-retail-retail-suite
```

### Шаг 9. Проверь runtime resources

```bash
kubectl get all -n env-retail-retail-suite
kubectl get configmap -n env-retail-retail-suite
kubectl get netpol -n env-retail-retail-suite
```

### Шаг 10. Проверь конкретные приложения

Например:

```bash
kubectl get deployment -n env-retail-retail-suite api -o yaml
kubectl get deployment -n env-retail-retail-suite admin -o yaml
```

Проверь:
- image
- replicas
- readyReplicas

### Шаг 11. Добавь следующую волну

Когда wave 1 зелёная:
- добавь ещё 3–4 приложения
- снова `validate`
- снова commit/push
- снова live-check

### Шаг 12. Добавь последние приложения

Повтори тот же цикл до всех 10.

## 10. Как проверить, что у каждого приложения применились правильные image и replicas

### Проверка image

```bash
kubectl get deployment -n env-retail-retail-suite api -o jsonpath='{.spec.template.spec.containers[0].image}'
```

Ожидаемый результат:

```text
ghcr.io/acme/api:1.4.2
```

### Проверка replicas

```bash
kubectl get deployment -n env-retail-retail-suite api -o jsonpath='{.spec.replicas}'
kubectl get deployment -n env-retail-retail-suite api -o jsonpath='{.status.readyReplicas}'
```

Оба значения должны совпасть с ожидаемым steady state.

### Быстрая массовая проверка

```bash
kubectl get deployment -n env-retail-retail-suite
```

Или по одному:

```bash
for app in api admin frontend billing notifications gateway search backoffice reporting-api; do
  echo "=== $app ==="
  kubectl get deployment -n env-retail-retail-suite "$app" -o jsonpath='{.spec.template.spec.containers[0].image}{" | replicas="}{.spec.replicas}{" | ready="}{.status.readyReplicas}{"\n"}'
done
```

## 11. Как проверить environment variables

Если `configOverrides` попали в `ConfigMap`, удобно смотреть так:

```bash
kubectl get configmap -n env-retail-retail-suite api-config -o yaml
```

Если нужно посмотреть Pod env:

```bash
kubectl get deployment -n env-retail-retail-suite api -o yaml
```

Ищи:
- `envFrom`
- `configMapRef`
- или прямой env wiring, в зависимости от runtime shape

### Что сравнивать

Сравни:
- ожидаемые ключи из таблицы
- реальные ключи в runtime

Например:
- `PORT`
- `LOG_LEVEL`
- `API_URL`

## 12. Как безопасно оставить часть приложений выключенными

Если не все 10 приложений готовы:

```yaml
- name: worker-ui
  type: webapp
  enabled: false
  imageRepository: ghcr.io/acme/worker-ui
  imageTag: 0.5.0
  replicas: 1
```

Это правильный способ:
- app уже описан
- app не создаётся в кластере
- позже можно просто включить `enabled: true`

## 13. Как обновлять уже добавленные приложения

Если через неделю нужно поменять только tag у `api`:

```yaml
- name: api
  type: webapp
  enabled: true
  imageRepository: ghcr.io/acme/api
  imageTag: 1.4.3
```

Дальше:

```bash
./cli/idp validate --file environments/retail/retail-suite.yaml
git add environments/retail/retail-suite.yaml
git commit -m "Update api image tag in retail-suite"
git push origin feature/phase-1-foundation
```

Если нужно поменять replicas:

```yaml
replicas: 3
```

## 14. Типичные ошибки

### Ошибка 1. Использовать одинаковые `name`

Плохо:

```yaml
- name: api
...
- name: api
```

Это будет rejected validation'ом.

### Ошибка 2. Пихать tag в `imageRepository`

Плохо:

```yaml
imageRepository: ghcr.io/acme/api:1.4.2
imageTag: latest
```

Правильно:

```yaml
imageRepository: ghcr.io/acme/api
imageTag: 1.4.2
```

### Ошибка 3. Коммитить секреты в `configOverrides`

Нельзя:

```yaml
configOverrides:
  DB_PASSWORD: supersecret
```

### Ошибка 4. Добавить сразу все 10 и потом не понимать, кто упал

Если сервисы новые и непроверенные, лучше вводить волнами.

### Ошибка 5. Ставить слишком большие replicas без необходимости

Для test environment часто хватает:
- `1`
- `2`

Не надо без причины стартовать всё по `3–5` replicas.

## 15. Готовый шаблон для копирования

Ниже blank-template, который удобно размножать руками.

```yaml
- name: <app-name>
  type: webapp
  enabled: true
  imageRepository: <registry/org/app>
  imageTag: <tag>
  replicas: 1
  configOverrides:
    PORT: "8080"
```

## 16. Рекомендуемый rollout plan для 10 приложений

Практически я бы рекомендовал так:

### Wave 1

- `api`
- `admin`

Цель:
- проверить базовый path
- проверить image pulls
- проверить env wiring

### Wave 2

- `frontend`
- `gateway`
- `search`

Цель:
- проверить multi-app environment

### Wave 3

- `billing`
- `notifications`
- `backoffice`
- `reporting-api`

Цель:
- довести основной комплект до полного набора

### Wave 4

- `worker-ui` или другие сомнительные приложения

Цель:
- включать только после подтверждения readiness

## 17. Минимальный чеклист перед каждым push

Перед каждым push проверь:

- все `name` уникальны
- все `type` = `webapp`
- `imageRepository` без tag
- `imageTag` указан отдельно
- `replicas` в диапазоне `0–50`
- в `configOverrides` нет секретов
- `enabled: false` стоит там, где app пока не должен запускаться
- `./cli/idp validate --file ...` проходит

## 18. Минимальный чеклист после каждого push

После push проверь:

- generated Argo Application есть
- `Environment` claim `Ready=True`
- namespace создан
- есть `Deployment` для каждого включённого приложения
- `spec.replicas` совпадает с ожидаемым
- `readyReplicas` вышел в steady state
- image соответствует ожидаемому tag

## 19. Когда всё-таки нужен другой путь

Этот guide НЕ подходит, если хотя бы одно из 10 приложений:
- не должно быть `Service`-ориентированным webapp
- требует `CronJob`
- требует queue/broker semantics
- требует отдельного provider или special runtime contract

Тогда уже смотри:
- [ADDING_NEW_PLATFORM_COMPONENT_TYPE.md](ADDING_NEW_PLATFORM_COMPONENT_TYPE.md)

## 20. Краткий summary

Для твоего кейса путь простой:

1. собрать таблицу по 10 приложениям
2. внести их в один или несколько environment manifests как `type: webapp`
3. у каждого задать `name`, `imageRepository`, `imageTag`, `replicas`, `configOverrides`
4. прогонять `validate`
5. выкатывать лучше волнами, а не всеми десятью сразу
6. после каждого push проверять Argo Application, Environment claim, namespace и Deployments

Это не platform-extension задача, а обычная GitOps-конфигурация на уже существующем type `webapp`.
