# Backstage Operations Guide

## Назначение

Этот документ описывает, какие операции в этой платформе можно и нужно делать
через Backstage, а какие лучше делать через Git или `kubectl`.

Документ ориентирован на практическое использование:
- куда зайти
- что нажать
- что ожидать
- что проверять после действия

## See Also

- [TESTING_GUIDE.md](TESTING_GUIDE.md)
- [USER_JOURNEYS.md](USER_JOURNEYS.md)
- [ADDING_MULTIPLE_WEBAPPS.md](ADDING_MULTIPLE_WEBAPPS.md)
- [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)
- [../OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)

## 0. Вход в Backstage (OIDC логин)

### Требования перед первым входом

В `/etc/hosts` браузера (или рабочей машины пользователя) должны быть:

```
89.108.100.41   backstage.idp.local
89.108.100.218  authentik-server.authentik.svc.cluster.local
```

Обе строки нужны. Backstage делает OIDC redirect на Authentik по internal DNS-имени из метаданных (`authentik-server.authentik.svc.cluster.local`), поэтому браузер должен уметь резолвить это имя.

### Как войти

1. Открыть `http://backstage.idp.local:7007`
2. Нажать **"Sign in with OIDC"**
3. В popup: логин `akadmin`, пароль `Admin1234!`
4. После успешного логина popup закрывается и Backstage открывается

### Если кнопка не появляется / появляется "Enter as a Guest"

Очистить кэш браузера (или открыть в режиме инкогнито с отключённым кэшем в DevTools). JavaScript-файл с логикой логина кэшируется браузером на 2 недели.

### OIDC архитектурные особенности этой установки

- Backstage 1.48.0 (prebuilt image) имеет sign-in page, захардкоженную в скомпилированном JS. Конфигурация через `app.signInPage: oidc` в app-config **не работает** в этом образе.
- OIDC логин реализован через патч `module-backstage.oidcpatch.js` в Dockerfile: guest Component заменён на OIDC popup, B-loader — на проверку сессии через `/api/auth/oidc/refresh`.
- Authentik использует `prompt: select_account`. **Не использовать `prompt: login`** — вызывает бесконечный цикл переаутентификации.
- Resolver: `emailMatchingUserEntityProfileEmail` — User entity `akadmin` (email `root@example.com`) должна присутствовать в Backstage catalog.

---

## 1. Что Backstage делает в этой платформе

Backstage здесь полезен как:

1. self-service UI
2. software catalog
3. live status entry point
4. migration dashboard
5. runbook/docs portal

Но Backstage не заменяет:
- Git как source of truth
- ArgoCD как reconciliation engine
- Crossplane как provisioning engine
- `kubectl` как low-level operational tool

## 2. Что лучше делать через Backstage, а что нет

### Хорошо делать через Backstage

- создать новый environment с типовым набором параметров
- найти environment в каталоге
- посмотреть owner/team metadata
- открыть live Kubernetes status
- посмотреть migration progress
- открыть runbooks и docs

### Лучше делать через Git

- массово редактировать environment manifest
- добавлять сразу 10 разных `webapp`-компонентов
- делать большие структурные изменения в `spec.components[]`
- review/PR workflow

### Лучше делать через `kubectl` / Argo

- разбирать reconcile failure
- проверять namespace/resources
- проверять точный runtime image
- проверять pod events/logs
- докапываться до конкретной причины, почему deployment не поднялся

## 3. Backstage маршруты, которые реально важны

В этой платформе для пользователя важны такие URL:

- `/`
  - главная страница Backstage
- `/catalog`
  - каталог сущностей
- `/create`
  - scaffolder templates
- `/migration`
  - migration dashboard
- `/docs`
  - runbooks/docs portal

### Практический пример

Если ты открыл локально:

```text
http://localhost:7007
```

то полезные страницы будут:

```text
http://localhost:7007/catalog
http://localhost:7007/create
http://localhost:7007/migration
http://localhost:7007/docs
```

## 4. Что можно сделать через Backstage в сценарии с environments

### Операция 1. Создать новый environment

Это основная user-facing операция.

Используется template:
- [../templates/new-environment/template.yaml](../templates/new-environment/template.yaml)

Что делает Backstage:
- показывает форму
- собирает typed parameters
- рендерит environment manifest
- рендерит catalog entry
- делает commit в GitOps branch
- регистрирует сущность в catalog

Что это даёт:
- не нужно вручную писать YAML с нуля
- меньше шанс ошибиться в обязательных полях

### Операция 2. Найти environment в catalog

Через catalog можно:
- увидеть, что environment зарегистрирован
- проверить owner/team metadata
- открыть entity page

### Операция 3. Посмотреть live Kubernetes status

Если catalog entity настроен правильно, можно открыть Kubernetes view и увидеть:
- deployments
- pods
- services
- readiness state

Это особенно полезно, когда:
- environment уже создан
- нужно быстро понять, жив он или нет
- не хочется сразу идти в `kubectl`

### Операция 4. Посмотреть migration dashboard

Маршрут:

```text
/migration
```

Что показывает:
- teams
- total
- completed
- validated
- in progress
- blocked
- percentage

### Операция 5. Открыть runbooks и docs

Маршрут:

```text
/docs
```

Что полезно:
- migration runbooks
- application-type specific guidance
- fallback page для missing runbook

## 5. Что нельзя или неудобно делать через текущий Backstage

Важно: текущий `new-environment` template покрывает типовой creation flow, но
он не является bulk-editor для сложного environment manifest.

### Неудобно через текущий template

- добавить 10 разных `webapp`-компонентов с разными image/env settings
- массово менять уже существующий большой manifest
- редактировать сложный `configOverrides` для множества приложений

Причина:
- текущий template параметризован под базовый environment creation flow
- он не является полноценным visual editor для больших `components[]`

### Практический вывод

Для твоего сценария:
- создать стартовый environment можно через Backstage
- но bulk-настройку 10 приложений лучше делать в Git

## 6. Подробно: как создать environment через Backstage

### Шаг 1. Открыть страницу создания

URL:

```text
http://localhost:7007/create
```

Ожидаемый экран:
- список scaffolder templates
- среди них шаблон `New Environment`

Что должно быть видно:
- title: `New Environment`
- описание про создание IDP test environment

### Шаг 2. Открыть template `New Environment`

После клика должна открыться форма с параметрами.

Ключевые поля из текущего template:
- `environment_name`
- `team`
- `owner`
- `webapp_enabled`
- `webapp_image_tag`
- `webapp_replicas`
- `postgresql_enabled`
- `redis_enabled`
- `redis_replicas`

### Шаг 3. Заполнить форму

Пример валидного ввода:

- `environment_name`: `commerce-smoke`
- `team`: `commerce`
- `owner`: `group:default/platform`
- `webapp_enabled`: `true`
- `webapp_image_tag`: `1.27.4`
- `webapp_replicas`: `2`
- `postgresql_enabled`: `true`
- `redis_enabled`: `false`
- `redis_replicas`: `1`

### Шаг 4. Submit

После submit Backstage запускает scaffolder task.

Ожидаемый результат:
- появится task execution page
- будут видны шаги render/publish/register

### Шаг 5. Проверить output task

На task page полезно смотреть:
- render environment manifest
- render catalog entry
- commit files to GitOps repository
- register catalog entry

Если всё прошло:
- должен появиться commit hash
- должна появиться ссылка на catalog entity

## 7. Что именно Backstage коммитит

Текущий template пишет:

- environment manifest:
  - `environments/<team>/<environment_name>.yaml`
- catalog entry:
  - `catalog/environments/<team>-<environment_name>.yaml`

Это важно понимать:
- Backstage не деплоит в кластер напрямую
- он делает Git commit
- дальше всё делает GitOps chain

## 8. Что проверить после создания через Backstage

После успешного task run:

### Проверка 1. Git commit

Проверить, что в branch появился новый commit.

### Проверка 2. Generated Argo Application

Пример:

```bash
kubectl get application -n argocd commerce-commerce-smoke -o yaml
```

### Проверка 3. Environment claim

```bash
kubectl get environment -n crossplane-system commerce-smoke -o yaml
```

### Проверка 4. Namespace

```bash
kubectl get ns env-commerce-commerce-smoke
```

### Проверка 5. Runtime resources

```bash
kubectl get all -n env-commerce-commerce-smoke
```

## 9. Как использовать Backstage catalog

### Где открыть

URL:

```text
http://localhost:7007/catalog
```

### Что искать

Можно искать по:
- имени environment
- owner
- team

### Что каталог реально даёт

Каталог полезен для:
- discoverability
- ownership visibility
- навигации

Он не заменяет runtime-debugging, но помогает быстро понять:
- существует ли environment вообще
- зарегистрирован ли он
- кому он принадлежит

## 10. Как использовать Kubernetes tab / live status

Для появления вкладки Kubernetes необходимо выполнить **оба** условия:

**Условие 1:** `spec.type: service` в catalog entry. Типы `environment`, `website` и т.д. используют layout без Kubernetes-вкладки в prebuilt образе.

**Условие 2:** аннотация `backstage.io/kubernetes-label-selector` (или `backstage.io/kubernetes-id`). Функция `isKubernetesAvailable` проверяет именно эти аннотации. Наличия `kubernetes-cluster` и `kubernetes-namespace` **недостаточно** — вкладка не появится без label-selector или id.

Полный набор обязательных аннотаций:
```yaml
backstage.io/kubernetes-cluster: primary
backstage.io/kubernetes-namespace: env-<team>-<env>
backstage.io/kubernetes-label-selector: idp.platform.io/team
```

Если environment entity и annotations настроены корректно, через Backstage можно:
- открыть entity
- перейти к Kubernetes status
- увидеть deployments/pods/services

Это удобно для first-level проверки.

### Когда этого достаточно

Достаточно, если нужно:
- быстро понять, жив ли environment
- проверить, что появились pods/services
- убедиться, что reconcile вообще дошёл до runtime

### Когда уже нужен `kubectl`

Нужен `kubectl`, если:
- pod не Ready
- image pull failed
- CrashLoopBackOff
- reconcile завис
- нет namespace
- нет claim

## 11. Как использовать `/migration`

URL:

```text
http://localhost:7007/migration
```

Что полезно:
- team-by-team status
- общий процент миграции
- blocked teams/environments

Для платформенной команды это полезно как:
- обзор прогресса
- короткий management view
- быстрый сигнал “где есть хвосты”

## 12. Как использовать `/docs`

URL:

```text
http://localhost:7007/docs
```

Что там можно делать:
- открыть runbook по application type
- пройти migration steps
- проверить fallback для неизвестного типа

Это особенно полезно для:
- новых инженеров
- self-service migration
- стандартных operational действий

## 13. Практический сценарий: что делать через Backstage, а что через Git

### Сценарий: нужно завести 10 новых webapp-приложений

### Через Backstage хорошо:

- создать стартовый environment как каркас
- проверить, что environment появился в catalog
- посмотреть live status после deploy
- открыть docs/runbooks

### Через Git лучше:

- добавить 10 отдельных `components[]`
- задать разные `imageRepository`
- задать разные `imageTag`
- задать разные `replicas`
- задать разные `configOverrides`

### Через `kubectl` лучше:

- убедиться, что каждый Deployment поднялся
- проверить конкретные images
- проверить `readyReplicas`
- посмотреть events и crash reasons

## 14. Пример полного workflow с Backstage + Git

### Вариант 1. Полностью через Git

Подходит лучше всего для bulk-case.

Путь:
- подготовить YAML
- validate
- commit/push
- проверить Argo/claim/namespace

### Вариант 2. Через Backstage как стартовую точку

Путь:
1. через Backstage создать базовый environment
2. получить стартовый commit и environment file
3. дальше руками расширить manifest в Git до 10 `webapp`-ов
4. снова push
5. использовать Backstage catalog/status как обзорную панель

Этот вариант часто удобнее, если:
- хочешь быстро получить skeleton
- потом вручную доработать сложный manifest

## 15. Конкретные полезные примеры

### Пример A. Создать новый базовый environment

В форме:
- `environment_name`: `commerce-suite`
- `team`: `commerce`
- `owner`: `group:default/platform`
- `webapp_enabled`: `true`
- `webapp_image_tag`: `1.27.4`
- `webapp_replicas`: `1`
- `postgresql_enabled`: `false`
- `redis_enabled`: `false`

Что получится:
- стартовый environment manifest
- catalog entry
- Git commit

### Пример B. Потом расширить этот environment в Git

После scaffolder run:
- открываешь `environments/commerce/commerce-suite.yaml`
- заменяешь один `webapp` на 10 конкретных components

Для этого смотри:
- [ADDING_MULTIPLE_WEBAPPS.md](ADDING_MULTIPLE_WEBAPPS.md)

## 16. Что делать, если в Backstage не видно template

Проверь:
- `/create`
- наличие `New Environment`

Если template не виден:
- возможно, проблема в static catalog registration
- проверь template registration artifacts:
  - [../templates/new-environment/template.yaml](../templates/new-environment/template.yaml)
  - [../catalog/all-components.yaml](../catalog/all-components.yaml)
  - [../platform/backstage/app-config.catalog.yaml](../platform/backstage/app-config.catalog.yaml)

## 17. Что делать, если task в Backstage прошёл, а environment не появился

Проверять по цепочке:

1. task output
2. Git commit
3. generated Argo Application
4. `Environment` claim
5. namespace
6. runtime resources

Именно так, потому что Backstage отвечает только за front-door flow, а не за весь reconcile chain.

## 18. Скриншоты

В этой репозитории можно и нужно использовать точные URL и ожидаемые экраны,
но живые скриншоты не включены в документ по двум причинам:

1. текущая агентная сессия headless и не даёт надёжно снять браузерные экраны
2. UI может меняться в зависимости от live Backstage instance, auth state и данных каталога

Вместо этого используй такие визуальные якоря:

### Экран 1. `/create`

Что должно быть видно:
- список templates
- `New Environment`

### Экран 2. `New Environment` form

Что должно быть видно:
- поля `environment_name`, `team`, `owner`
- toggles для `webapp`, `postgresql`, `redis`

### Экран 3. scaffolder task page

Что должно быть видно:
- render step
- publish step
- register step
- commit hash

### Экран 4. `/catalog`

Что должно быть видно:
- environment entity
- owner/team metadata

### Экран 5. `/migration`

Что должно быть видно:
- table по командам
- counts / percentage

### Экран 6. `/docs`

Что должно быть видно:
- список runbooks
- fallback page для unknown runbook type

## 19. Краткий summary

Backstage в этой платформе нужен не как замена GitOps, а как:
- front door
- catalog
- status entry point
- docs/runbooks portal

Для твоего bulk-case с 10 `webapp`-ами лучший практический путь такой:

1. при желании создать skeleton через Backstage
2. основной bulk-edit сделать через Git
3. использовать Backstage для обзора и статуса
4. использовать `kubectl` для точной диагностики
