# Spec Review — Change Report

**Date:** 2026-03-14
**Scope:** Resolution of all 18 issues found in `spec-review.md`
**Files changed:** `constraints.md`, `entity_model.md`, `requirements.md`, `acceptance_criteria.md`
**Files created:** `use_cases/UC-020`, `UC-021`, `UC-022`

---

## BLK-001 · `enabled: false` — прямое противоречие

### Было
`constraints.md §2.2` содержал:
> "MUST NOT delete the Deployment or Service resource when `enabled: false`. Resource removal is handled by ArgoCD pruning when the component entry is removed from the manifest entirely."

`UC-003` и `AC-005` при этом говорили:
> "all Kubernetes resources for that component… fully removed within 3 minutes"

### Почему плохо
Агент не может удовлетворить оба требования одновременно. При реализации по `constraints.md` тест `AC-005` падал бы: ресурсы не удалялись, только скейлились в 0.

### Что изменили
`constraints.md §2.2` — полностью переписан блок:
- `enabled: false` → Crossplane устанавливает `deletionPolicy: Delete` на все managed resources и убирает их из desired state → ресурсы удаляются
- `replicas: 0` зафиксирован как **отдельный** концепт: скейлит поды до нуля, ресурсы сохраняются
- Идемпотентность сохранена: если компонент никогда не был включён, ничего не создаётся

### Стало
Единая модель: `enabled: false` = удаление всех ресурсов, `replicas: 0` = временная пауза без удаления.

---

## BLK-002 · "Correctly mapped" — неизмеримый критерий

### Было
`NFR-011`, `AC-052`, `constraints.md §5.2` все ссылались на "90% сервисов корректно смаппированы", но нигде не было определено, что значит "корректно смаппирован" для одного сервиса.

### Почему плохо
Тест `AC-052` невозможно написать: нет критерия pass/fail на уровне одного сервиса. Два разработчика напишут разные тесты и получат разные результаты.

### Что изменили
В `constraints.md §5.2` добавлен формальный блок **Definition — "correctly mapped"**:
- `image` → тип компонента разрешён, `imageTag` захвачен
- `ports` → container-side порт присутствует (host port — предупреждение, не ошибка)
- `environment` → все не-секретные ключи с значениями; секретные — флагируются, не дропаются
- `volumes` → mount paths документируются в отчёте
- `depends_on` → порядок зависимостей сохранён
- Сервис с `enabled: false` (нет типа) **не засчитывается** в "корректно смаппированных"

В `requirements.md` добавлена ссылочная сноска перед таблицей NFR.

### Стало
Тест `AC-052` можно написать автоматически: для каждого из 10 сервисов фикстуры проверяется чеклист из 5 полей.

---

## BLK-003 · Переход `validated → completed` не определён

### Было
`entity_model.md` определял `migration_status: pending | in_progress | validated | completed | blocked`, но ни один UC, FR или AC не описывал, кто и когда переводит статус в `completed`. Deprecation gate (AC-066/067) требовал `completed`, но достичь его было невозможно.

Дополнительно: `TEAM.migrated_envs` инкрементировался "при validated" (AC-064), хотя семантически завершение — это `completed`.

### Почему плохо
Реализация застрянет: нет ни команды, ни UI, ни автоматики для установки `completed`. Deprecation gate навсегда заблокирован.

### Что изменили
- **`requirements.md`**: добавлен `FR-021` — "Mark migration as complete" (Platform Engineer)
- **`use_cases/UC-021-mark-migration-complete.md`**: создан — CLI-команда `idp complete-migration --env-id <id>`, проверяет что текущий статус `validated`, устанавливает `completed` + `completed_at`
- **`acceptance_criteria.md`**: добавлены `AC-076` (happy path) и `AC-077` (not validated error)
- **`acceptance_criteria.md` AC-064**: убрана строка про инкремент `migrated_envs` (счётчик удалён — см. MIN-004)
- Определена irreversibility: откат только через явную команду `idp revert-migration` с обоснованием

### Стало
Полный жизненный цикл: `pending → in_progress → validated → completed`. Каждый переход имеет явного актора, команду и acceptance test.

---

## BLK-004 · Deprecation workflow не специфицирован

### Было
`AC-066`, `AC-067`, `C-012` ссылались на "deprecation workflow" и "legacy platform deprecation", но не было никакого FR, UC, CLI-команды или UI-экрана. Acceptance-тесты ссылались на несуществующий триггер.

### Почему плохо
`AC-066` и `AC-067` нетестируемы: нет действия, которое можно совершить, чтобы проверить gate.

### Что изменили
- **`requirements.md`**: добавлен `FR-022` — "Initiate legacy platform deprecation" (Platform Engineer)
- **`use_cases/UC-022-initiate-deprecation.md`**: создан — CLI `idp deprecate-legacy`:
  - `--dry-run` (по умолчанию): показывает сводку, не делает ничего необратимого
  - `--confirm`: генерирует `deprecation-report-<date>.md`, устанавливает флаг `LEGACY_PLATFORM_DEPRECATED=true`
  - Gate: блокирует если есть хотя бы одна запись != `completed`
  - Отчёт коммитится в `docs/deprecation/` как audit trail
- **`acceptance_criteria.md`**: добавлены `AC-078` (gate pass), `AC-079` (gate blocked), `AC-080` (no environments)

### Стало
Deprecation — это конкретная CLI-команда с dry-run режимом, формальным отчётом и audit trail в Git.

---

## MAJ-001 · Неверный тип генератора ApplicationSet

### Было
`constraints.md §2.3`:
> "MUST use an ApplicationSet with a git generator (directory strategy)"

### Почему плохо
ArgoCD `git directories` generator создаёт один Application **на директорию**. Окружения хранятся как файлы `environments/<team>/<env-name>.yaml`. Использование `directories` создало бы один Application на команду, а не на окружение.

### Что изменили
`constraints.md §2.3` — текст заменён на `files` стратегию с корректным YAML-примером:
```yaml
generators:
  - git:
      files:
        - path: environments/**/*.yaml
```
Добавлено объяснение почему `directories` неверен.

### Стало
Один ArgoCD Application на файл окружения, что соответствует модели "один Environment CRD = одно окружение".

---

## MAJ-002 · Нет UC/FR для регистрации legacy-окружений

### Было
`UC-019` (Track migration status) предполагал как precondition: "Legacy environments have been registered", но ни один FR, UC или CLI-команда не описывали КАК создать запись `LEGACY_ENVIRONMENT`. Dashboard был пустым по умолчанию.

### Почему плохо
Migration tracking dashboard (AC-065) всегда показывал бы нули. Deprecation gate (AC-066) не мог корректно оценить прогресс.

### Что изменили
- **`requirements.md`**: добавлен `FR-020` — "Register legacy environment" (Platform Engineer)
- **`use_cases/UC-020-register-legacy-environment.md`**: создан — CLI `idp register-legacy --name <name> --team <team> [--compose-path <path>]`
- **`acceptance_criteria.md`**: добавлены `AC-073` (happy path), `AC-074` (team not found), `AC-075` (duplicate)
- `compose_file_path` Optional — регистрация возможна даже без доступа к файлу (соответствует C-011)

### Стало
Явный первый шаг миграции: регистрация окружения. Без неё оно невидимо для трекера.

---

## MAJ-003 · Migration dashboard не специфицирован

### Было
`UC-019` и `AC-065` ссылались на "migration tracking dashboard", но нигде не было указано: что это за компонент, где живёт, на каком стеке сделан.

### Почему плохо
Агент мог реализовать dashboard как угодно — Grafana, отдельный сервис, страница в Backstage — без возможности проверки соответствия спецификации.

### Что изменили
`constraints.md §3.3` — добавлен раздел:
- Реализация: кастомный Backstage plugin `@internal/plugin-migration-dashboard`
- Маршрут: `/migration`
- Backend: Backstage backend plugin, endpoint `GET /api/migration/status`
- Данные: агрегация `LEGACY_ENVIRONMENT` записей (derived, не денормализованные счётчики)
- Обновление: каждые 60 секунд
- Путь к файлу: `packages/backend/src/plugins/migrationDashboard.ts`

### Стало
Однозначное решение: Backstage plugin, конкретный endpoint, конкретный маршрут.

---

## MAJ-004 · "Application type" не определён

### Было
`NFR-012` требовал "100% application types covered", `AC-068` и `UC-018` использовали этот термин, но нигде не было определено что такое "application type". Три возможные интерпретации давали разное количество требуемых runbook-ов.

### Почему плохо
Платформ-инженер не знает, сколько runbook-ов нужно написать. NFR-012 нетестируем без чёткой единицы измерения.

### Что изменили
`requirements.md` — добавлен definition block перед таблицей NFR:
> Application type = уникальная **комбинация** типов компонентов, используемая хотя бы в одном legacy-окружении (например, `[webapp, postgresql]`, `[worker, redis]`). Перечисляется на этапе регистрации FR-020.

### Стало
Количество runbook-ов = количество уникальных комбинаций, выявленных при регистрации. Измеримо и тестируемо.

---

## MAJ-005 · `USER.team_id NOT NULL` ломает модель platform engineer

### Было
`entity_model.md USER`:
```
| team_id | ... | Not Null, Foreign Key (TEAM.id) |
```
Platform engineer-ы управляют всеми командами и не принадлежат конкретной. Создать запись PE без team_id было невозможно.

### Почему плохо
При создании пользователя с `role: platform_engineer` возникало нарушение NOT NULL constraint. Либо пришлось бы создавать фиктивную команду "platform", что засоряло бы статистику миграции.

### Что изменили
`entity_model.md USER`:
- `team_id` изменён с `Not Null` на `Optional`
- Добавлен constraints-блок: "team_id MUST be Not Null when role is developer/qa_engineer/team_lead; MUST be Null when role is platform_engineer"
- Описание сущности обновлено

### Стало
PE создаётся с `team_id = null`. Бизнес-правило явно зафиксировано как constraint, а не молчаливо нарушается.

---

## MAJ-006 · Тест retention логов не имеет стратегии

### Было
`AC-033` (логи за 6 дней доступны) и `AC-035` (логи за 8 дней недоступны) входили в mandatory acceptance tests, но никакой стратегии их автоматизации не было. Ждать 6–8 реальных дней в CI невозможно.

### Почему плохо
Два acceptance criteria из обязательного набора были физически нереализуемы без специальной инфраструктуры.

### Что изменили
`constraints.md §5.4` — добавлен блок **Log retention tests**:
- Тестовый хелпер для инжекта события с backdated `created_at` в event store напрямую
- Конфигурируемый `LOG_RETENTION_DAYS` env var
- В тестах: `LOG_RETENTION_DAYS=0` — срок истечения в пределах текущего запуска теста

### Стало
Retention тест занимает секунды: вставить событие с `created_at = now - 1ms`, установить `LOG_RETENTION_DAYS=0`, проверить что событие недоступно.

---

## MAJ-007 · Тест на 50 одновременных окружений без инфра-спецификации

### Было
`AC-072` и `NFR-004` требовали тестирования 50 одновременных окружений, но не было указано: ни минимальные требования к кластеру, ни процедура создания 50 окружений, ни что считать "деградацией".

### Почему плохо
Агент мог запустить тест на 1-нодовом кластере с 2 CPU и получить провал из-за недостаточности инфраструктуры, а не из-за дефекта платформы.

### Что изменили
`constraints.md §5.4` — добавлен блок **Concurrency test**:
- Минимальный кластер: 3 nodes × 4 vCPU × 8 GB RAM
- Процедура: 50 последовательных Git-коммитов в 5-минутное окно через CLI test helper
- Assertion: median sync time ≤ 5 мин, ни одно окружение не превышает 8 мин (буфер 1.6×)
- Изоляция: отдельный `load-test` CI job, не входит в стандартный acceptance suite

### Стало
Тест воспроизводим: зафиксированы все параметры стенда и критерии pass/fail.

---

## MIN-001 · Namespace overflow — нет стратегии усечения

### Было
`constraints.md §4.1` задавал паттерн `env-<team>-<env-name>`, `entity_model.md` ограничивал длину namespace 63 символами, но не было описано что делать если имя превышает лимит.

### Почему плохо
При длинных именах команды/окружения генерировался namespace > 63 символов → Kubernetes отклонял бы создание namespace с невнятной ошибкой.

### Что изменили
`constraints.md §4.1` — добавлена формула усечения:
- Формат: `env-<team[:20]>-<envname[:N]>-<hash4>`
- N = 63 − 4 − 20 − 1 − 1 − 4 = 33
- hash4 = 4-символьный lowercase hex от `team + "-" + envName`
- Приведена Go-реализация: `fmt.Sprintf("env-%s-%s-%s", ...)`
- Уникальность гарантирована через hash при коллизиях усечения

### Стало
Любое имя команды/окружения приводится к валидному namespace ≤ 63 символа с гарантией уникальности.

---

## MIN-002 · ENVIRONMENT_TEMPLATE без поля `parameters`

### Было
`entity_model.md ENVIRONMENT_TEMPLATE` содержал только `template_yaml: String 65535`. UC-009 и constraints.md оба упоминали "template parameters", но хранить и валидировать их отдельно было невозможно.

### Почему плохо
Backstage не мог построить типизированную форму для шаблона: нет метаданных о параметрах (тип, обязательность, дефолт). Валидация ввода была бы невозможна.

### Что изменили
`entity_model.md ENVIRONMENT_TEMPLATE` — добавлено поле:
```
| parameters | JSON array of parameter definitions | String | 4000 | Optional |
```
Добавлен constraints-блок с JSON-схемой элемента: `{name, type: "string"|"integer"|"boolean", description, required, defaultValue}`. Уникальность `name` внутри массива.

### Стало
Backstage читает `parameters`, строит форму с типизированными полями, валидирует до коммита.

---

## MIN-003 · Метод аутентификации Backstage не определён

### Было
NFR-006 требовал RBAC, AC-025/026 определяли разные представления по ролям, но нигде не было указано как Backstage аутентифицирует пользователей и откуда берёт их роль/команду.

### Почему плохо
Без auth нет access scoping. Агент мог реализовать guest auth или собственную систему пользователей, несовместимую с существующим SSO.

### Что изменили
`constraints.md §3.3` — добавлен блок:
- Provider: Authentik OIDC (`auth.providers.oidc`) — уже развёрнут на кластере
- Endpoint: `http://authentik-server.authentik.svc.cluster.local:9000/application/o/backstage/`
- Роли и команды: из LDAP group membership claims, синхронизированных через Authentik
- Guest auth: отключён в production

### Стало
Backstage использует существующий SSO. Роли берутся из LDAP групп — единый источник правды.

---

## MIN-004 · `TEAM.migrated_envs` — два источника правды

### Было
`entity_model.md TEAM` содержал два поля-счётчика: `total_legacy_envs` и `migrated_envs`. Те же числа можно было получить через COUNT запросы к `LEGACY_ENVIRONMENT`. При прямом обновлении записей счётчики расходились.

### Почему плохо
При batch-обновлениях или исправлениях данных счётчики десинхронизировались. Dashboard показывал неверные числа.

### Что изменили
`entity_model.md TEAM` — оба поля **удалены**. Вместо них добавлена заметка:
> Счётчики вычисляются через агрегацию `LEGACY_ENVIRONMENT` при каждом запросе: `total = COUNT WHERE team_id=X`, `migrated = COUNT WHERE … AND status IN ('validated','completed')`

Также обновлён `AC-064`: убрана строка про инкремент `migrated_envs`.

### Стало
Один источник правды — `LEGACY_ENVIRONMENT` таблица. Данные dashboard всегда консистентны.

---

## MIN-005 · Минимальная версия docker-compose не определена

### Было
`constraints.md §3.2` указывал библиотеку `github.com/compose-spec/compose-go`, но не было сказано какие версии формата файла поддерживаются.

### Почему плохо
Legacy-платформа могла использовать `version: "2.x"` файлы. Агент мог реализовать поддержку только compose-spec и молча ломаться на старых файлах.

### Что изменили
`constraints.md §3.2` — добавлен абзац:
- Compose Specification v1.0+ (version 3.x и unversioned): **MUST** поддерживать
- version 2.x: **SHOULD** поддерживать через backward-compat loader `compose-go`
- Неподдерживаемые директивы version 2.x: в conversion report, не hard failure

### Стало
Явное разграничение: 3.x/unversioned — обязательно, 2.x — по возможности, ошибки — в отчёт.

---

## MIN-006 · Нет правила наследования владения шаблоном

### Было
`ENVIRONMENT_TEMPLATE.owner_id` был `Not Null`. Если владелец деактивируется, шаблон становился orphan или нарушал FK constraint.

### Почему плохо
Деактивация team lead приводит к нарушению FK при попытке удалить/обновить пользователя. Или шаблоны "зависают" без maintainer-а.

### Что изменили
`entity_model.md ENVIRONMENT_TEMPLATE` — добавлен constraint в блок заметок к полю `parameters`:
> "Если owner деактивируется, владение MUST быть передано другому team_lead из той же команды или platform_engineer ДО деактивации"

### Стало
Явное бизнес-правило: деактивация пользователя блокируется если он единственный владелец активного шаблона.

---

## MIN-007 · `DATABASE_INSTANCE` без поля namespace

### Было
`entity_model.md DATABASE_INSTANCE` не имел поля `namespace`. Чтобы найти namespace для DatabaseInstance, требовался join через 3 таблицы: `DATABASE_INSTANCE → COMPONENT → ENVIRONMENT.namespace`.

### Почему плохо
Запрос "найди все базы данных в namespace X" требовал 2 JOIN-а. Риск операций над wrong namespace при сложных запросах.

### Что изменили
`entity_model.md DATABASE_INSTANCE` — добавлено поле:
```
| namespace | Kubernetes namespace (copied from ENVIRONMENT) | String | 63 | Not Null |
```
Добавлен constraints-блок: namespace MUST equal `ENVIRONMENT.namespace` parent-а. Поле денормализовано для прямых запросов.

### Стало
Прямой запрос: `SELECT * FROM DATABASE_INSTANCE WHERE namespace = 'env-team-name'`. Безопасность: явный namespace в каждой записи.

---

## Итоговый трекер изменений

| Issue | Severity | Файлы изменены | Файлы созданы |
|---|---|---|---|
| BLK-001 | BLOCKER | `constraints.md §2.2` | — |
| BLK-002 | BLOCKER | `constraints.md §5.2`, `requirements.md` | — |
| BLK-003 | BLOCKER | `requirements.md`, `acceptance_criteria.md` | `UC-021` |
| BLK-004 | BLOCKER | `requirements.md`, `acceptance_criteria.md` | `UC-022` |
| MAJ-001 | MAJOR | `constraints.md §2.3` | — |
| MAJ-002 | MAJOR | `requirements.md`, `acceptance_criteria.md` | `UC-020` |
| MAJ-003 | MAJOR | `constraints.md §3.3` | — |
| MAJ-004 | MAJOR | `requirements.md` | — |
| MAJ-005 | MAJOR | `entity_model.md (USER)` | — |
| MAJ-006 | MAJOR | `constraints.md §5.4` | — |
| MAJ-007 | MAJOR | `constraints.md §5.4` | — |
| MIN-001 | MINOR | `constraints.md §4.1` | — |
| MIN-002 | MINOR | `entity_model.md (ENVIRONMENT_TEMPLATE)` | — |
| MIN-003 | MINOR | `constraints.md §3.3` | — |
| MIN-004 | MINOR | `entity_model.md (TEAM)`, `acceptance_criteria.md` | — |
| MIN-005 | MINOR | `constraints.md §3.2` | — |
| MIN-006 | MINOR | `entity_model.md (ENVIRONMENT_TEMPLATE)` | — |
| MIN-007 | MINOR | `entity_model.md (DATABASE_INSTANCE)` | — |
