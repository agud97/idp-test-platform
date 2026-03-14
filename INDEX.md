# Documentation Index

## See Also

- [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md)
- [HANDOFF.md](HANDOFF.md)
- [OPERATIONS_CHECKLIST.md](OPERATIONS_CHECKLIST.md)
- [docs/status.md](docs/status.md)
- [docs/POST_IMPLEMENTATION_DOC_CHANGES.md](docs/POST_IMPLEMENTATION_DOC_CHANGES.md)
- [docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md](docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md)
- [SESSION_BOOTSTRAP.md](SESSION_BOOTSTRAP.md)
- [docs/ADDING_NEW_PLATFORM_COMPONENT_TYPE.md](docs/ADDING_NEW_PLATFORM_COMPONENT_TYPE.md)
- [docs/NEW_COMPONENT_TYPE_PR_TEMPLATE.md](docs/NEW_COMPONENT_TYPE_PR_TEMPLATE.md)
- [docs/ADDING_MULTIPLE_WEBAPPS.md](docs/ADDING_MULTIPLE_WEBAPPS.md)
- [docs/BACKSTAGE_OPERATIONS_GUIDE.md](docs/BACKSTAGE_OPERATIONS_GUIDE.md)
- [TESTING_GUIDE.md](docs/TESTING_GUIDE.md)
- [USER_JOURNEYS.md](docs/USER_JOURNEYS.md)

## Назначение

Этот файл — точка входа в эксплуатационную и итоговую документацию репозитория.

Если непонятно, какой документ читать первым, начинай отсюда.

## Быстрый выбор документа

### Нужна общая картина релиза

Читай [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md).

Когда использовать:
- нужно понять, что именно реализовано
- нужно увидеть финальный scope по всему репозиторию
- нужно быстро понять, что было реально провалидировано
- нужно показать summary руководителю или следующему инженеру

Что внутри:
- high-level итог по фазам
- список готовых subsystem'ов
- acceptance/security/load validation
- canonical re-check commands
- residual risks

### Нужна передача платформы следующему инженеру

Читай [HANDOFF.md](HANDOFF.md).

Когда использовать:
- onboarding нового инженера
- передача дежурства
- подготовка к первой эксплуатации
- нужен список того, что должно быть настроено и что мониторить

Что внутри:
- что запускать в первый день эксплуатации
- какие секреты и переменные нужны
- что мониторить после релиза
- какие operational команды считать canonical
- какие риски и осторожности учитывать

### Нужны практические ежедневные команды и процедуры

Читай [OPERATIONS_CHECKLIST.md](OPERATIONS_CHECKLIST.md).

Когда использовать:
- ежедневный health check
- разбор конкретного environment
- повторная live-валидация после изменений
- incident triage
- проверка GitOps / Argo / Crossplane / Backstage / CLI

Что внутри:
- пошаговые operational checks
- команды и интерпретация результатов
- диагностические сценарии
- минимальный рабочий набор проверок

### Нужен подробный guide по использованию платформы

Читай [TESTING_GUIDE.md](docs/TESTING_GUIDE.md).

Когда использовать:
- нужно понять полный workflow миграции из `docker-compose`
- нужно запустить test environment через GitOps
- нужно разобраться, как добавлять новые сервисы
- нужен более обучающий документ, а не только checklist

Что внутри:
- подробные команды и объяснения
- migration workflow
- environment lifecycle workflow
- добавление новых сервисов и component types

Если нужен именно отдельный deep-dive по platform engineering сценарию
добавления нового type:
- читай [ADDING_NEW_PLATFORM_COMPONENT_TYPE.md](docs/ADDING_NEW_PLATFORM_COMPONENT_TYPE.md)

Если нужен готовый шаблон PR под такой change:
- читай [NEW_COMPONENT_TYPE_PR_TEMPLATE.md](docs/NEW_COMPONENT_TYPE_PR_TEMPLATE.md)

Если нужно массово добавить много новых приложений, которые уже подходят под
`type: webapp`:
- читай [ADDING_MULTIPLE_WEBAPPS.md](docs/ADDING_MULTIPLE_WEBAPPS.md)

Если нужен отдельный практический guide по тому, что именно можно делать через
Backstage и как это использовать вместе с GitOps:
- читай [BACKSTAGE_OPERATIONS_GUIDE.md](docs/BACKSTAGE_OPERATIONS_GUIDE.md)

### Нужны типовые сценарии от лица пользователя

Читай [USER_JOURNEYS.md](docs/USER_JOURNEYS.md).

Когда использовать:
- onboarding
- demos
- handoff продуктовой или платформенной команде
- быстрый walkthrough без чтения всей архитектуры

Что внутри:
- несколько end-to-end сценариев
- ожидаемые шаги и результаты
- куда смотреть при проблемах на каждом этапе

### Нужен фактический прогресс исполнения плана

Читай [docs/status.md](docs/status.md).

Когда использовать:
- нужно понять, какие задачи были закрыты
- нужен trace task-by-task
- нужно увидеть approved deviations и blockers history

Если нужно понять, какие исходные spec/plan формулировки были переписаны после реального исполнения:
- читай [docs/POST_IMPLEMENTATION_DOC_CHANGES.md](docs/POST_IMPLEMENTATION_DOC_CHANGES.md)

Если нужно понять, что следующий агент уже не должен расследовать заново, а что всё ещё остаётся внешним риском среды:
- читай [docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md](docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md)

Если нужна короткая стартовая памятка для любой новой сессии:
- читай [SESSION_BOOTSTRAP.md](SESSION_BOOTSTRAP.md)

## Рекомендуемый порядок чтения

### Для руководителя / reviewer

1. [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md)
2. [docs/status.md](docs/status.md)

### Для инженера, который принимает систему

1. [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md)
2. [HANDOFF.md](HANDOFF.md)
3. [OPERATIONS_CHECKLIST.md](OPERATIONS_CHECKLIST.md)

### Для инженера on-call / operations

1. [OPERATIONS_CHECKLIST.md](OPERATIONS_CHECKLIST.md)
2. [HANDOFF.md](HANDOFF.md)

### Для разработчика, который меняет platform code

1. [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md)
2. [OPERATIONS_CHECKLIST.md](OPERATIONS_CHECKLIST.md)
3. [docs/status.md](docs/status.md)

## Карта документов

- [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md)
  - итоговая release-level картина
- [HANDOFF.md](HANDOFF.md)
  - practical handoff и эксплуатационный контекст
- [OPERATIONS_CHECKLIST.md](OPERATIONS_CHECKLIST.md)
  - команды, проверки и operational troubleshooting
- [TESTING_GUIDE.md](docs/TESTING_GUIDE.md)
  - подробный guide по тестированию и практическому использованию
- [docs/ADDING_NEW_PLATFORM_COMPONENT_TYPE.md](docs/ADDING_NEW_PLATFORM_COMPONENT_TYPE.md)
  - отдельный пошаговый guide по добавлению нового platform component type
- [docs/NEW_COMPONENT_TYPE_PR_TEMPLATE.md](docs/NEW_COMPONENT_TYPE_PR_TEMPLATE.md)
  - готовый PR template для изменений такого типа
- [docs/ADDING_MULTIPLE_WEBAPPS.md](docs/ADDING_MULTIPLE_WEBAPPS.md)
  - отдельный практический guide по массовому добавлению webapp-компонентов
- [docs/BACKSTAGE_OPERATIONS_GUIDE.md](docs/BACKSTAGE_OPERATIONS_GUIDE.md)
  - отдельный guide по Backstage-операциям и их месту в общем workflow
- [USER_JOURNEYS.md](docs/USER_JOURNEYS.md)
  - типовые end-to-end пользовательские сценарии
- [docs/status.md](docs/status.md)
  - формальная история исполнения плана
- [docs/POST_IMPLEMENTATION_DOC_CHANGES.md](docs/POST_IMPLEMENTATION_DOC_CHANGES.md)
  - что и почему было переписано в нормативных docs после исполнения
- [docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md](docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md)
  - какие blockers уже сняты, а какие риски останутся на новом кластере
- [SESSION_BOOTSTRAP.md](SESSION_BOOTSTRAP.md)
  - короткий обязательный bootstrap для следующей сессии

## Если нужно начать с одного файла

Выбор такой:
- нужен обзор: [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md)
- нужно эксплуатировать: [OPERATIONS_CHECKLIST.md](OPERATIONS_CHECKLIST.md)
- нужно передать систему: [HANDOFF.md](HANDOFF.md)
- нужно научиться пользоваться: [TESTING_GUIDE.md](docs/TESTING_GUIDE.md)
- нужны готовые сценарии: [USER_JOURNEYS.md](docs/USER_JOURNEYS.md)
