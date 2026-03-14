# Documentation Sitemap

## Назначение

Этот документ — иерархическая карта всей документации проекта.

Он нужен, когда хочется увидеть не просто список файлов, а структуру:
- какой документ за что отвечает
- где искать обзор, а где runbook
- как документы соотносятся друг с другом

## Top-Level Entry Points

- [README.md](../README.md)
  - краткая корневая точка входа
  - основные ссылки и canonical validation commands

- [INDEX.md](../INDEX.md)
  - главный навигационный файл
  - помогает быстро выбрать нужный документ по задаче

## Release / Delivery Documents

- [RELEASE_SUMMARY.md](../RELEASE_SUMMARY.md)
  - release-style итог по всему репозиторию
  - что реализовано, что провалидировано, какие команды считать canonical

- [HANDOFF.md](../HANDOFF.md)
  - practical handoff
  - что передавать следующему инженеру
  - какие переменные, секреты и контрольные точки важны

- [docs/status.md](status.md)
  - фактическая история исполнения плана
  - task-by-task progress
  - deviations, blockers, checkpoint history

## Architecture / Design Documents

- [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)
  - технический обзор всей платформы
  - GitOps, ArgoCD, Crossplane, Backstage, CLI, testing layers

- [GLOSSARY.md](GLOSSARY.md)
  - термины и определения
  - помогает быстро снять путаницу между claim/composite/appset/provider и т.д.

## Operations / Runtime Documents

- [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)
  - ежедневные проверки
  - пошаговые operational команды
  - диагностика по слоям платформы

- [TROUBLESHOOTING_MATRIX.md](TROUBLESHOOTING_MATRIX.md)
  - быстрый путь от симптома к вероятной причине
  - полезен во время incident/debug

- [FAQ.md](FAQ.md)
  - короткие практические ответы на частые вопросы
  - удобен как первый слой для типовых ситуаций

## Usage / Workflow Documents

- [TESTING_GUIDE.md](TESTING_GUIDE.md)
  - подробный guide по использованию платформы
  - acceptance/security/load tests
  - migration из `docker-compose`
  - запуск test environments
  - добавление новых сервисов и component types

- [USER_JOURNEYS.md](USER_JOURNEYS.md)
  - типовые end-to-end пользовательские сценарии
  - полезен для onboarding, demos и handoff

## Original Spec / Plan Documents

Ниже — исходный spec-driven набор, на котором строилась реализация.

- [requirements.md](requirements.md)
  - функциональные требования, NFR, constraints references

- [acceptance_criteria.md](acceptance_criteria.md)
  - WHEN-THEN-SHALL критерии

- [constraints.md](constraints.md)
  - архитектурные правила и ограничения

- [entity_model.md](entity_model.md)
  - модель сущностей

- [vision.md](vision.md)
  - общее видение проекта

- [use_cases.md](use_cases.md)
  - overview use cases

- [use_cases.puml](use_cases.puml)
  - PlantUML диаграмма use cases

- [use_cases/](use_cases)
  - детальные use case документы

- [plan.yaml](plan.yaml)
  - исходный execution plan

- [spec-review.md](spec-review.md)
  - findings по исходной спецификации

- [spec-review-changes.md](spec-review-changes.md)
  - принятые исправления спецификации

## Recommended Reading Paths

### Если ты новый инженер

1. [README.md](../README.md)
2. [INDEX.md](../INDEX.md)
3. [RELEASE_SUMMARY.md](../RELEASE_SUMMARY.md)
4. [HANDOFF.md](../HANDOFF.md)
5. [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)

### Если ты on-call / operations engineer

1. [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)
2. [TROUBLESHOOTING_MATRIX.md](TROUBLESHOOTING_MATRIX.md)
3. [FAQ.md](FAQ.md)

### Если ты хочешь понять архитектуру

1. [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)
2. [GLOSSARY.md](GLOSSARY.md)
3. [RELEASE_SUMMARY.md](../RELEASE_SUMMARY.md)

### Если ты хочешь пользоваться платформой руками

1. [TESTING_GUIDE.md](TESTING_GUIDE.md)
2. [USER_JOURNEYS.md](USER_JOURNEYS.md)
3. [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)

### Если ты хочешь проверить, как всё было реализовано по плану

1. [docs/status.md](status.md)
2. [plan.yaml](plan.yaml)
3. [RELEASE_SUMMARY.md](../RELEASE_SUMMARY.md)

## Short Navigation Cheat Sheet

- нужен overview: [RELEASE_SUMMARY.md](../RELEASE_SUMMARY.md)
- нужен handoff: [HANDOFF.md](../HANDOFF.md)
- нужна эксплуатация: [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)
- нужен troubleshooting: [TROUBLESHOOTING_MATRIX.md](TROUBLESHOOTING_MATRIX.md)
- нужен usage guide: [TESTING_GUIDE.md](TESTING_GUIDE.md)
- нужны user scenarios: [USER_JOURNEYS.md](USER_JOURNEYS.md)
- нужен словарь: [GLOSSARY.md](GLOSSARY.md)
- нужна короткая Q&A: [FAQ.md](FAQ.md)
