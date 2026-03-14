# IDP Test Environment Platform

Репозиторий содержит реализацию GitOps-управляемой платформы для тестовых environments с:
- Crossplane provisioning
- ArgoCD reconciliation
- Backstage portal
- CLI migration tool для legacy environments
- acceptance, security и load validation

## Быстрые ссылки

- [INDEX.md](INDEX.md)
  - главная точка входа в документацию
- [RELEASE_SUMMARY.md](RELEASE_SUMMARY.md)
  - итоговый release-style summary
- [HANDOFF.md](HANDOFF.md)
  - practical handoff для следующего инженера
- [OPERATIONS_CHECKLIST.md](OPERATIONS_CHECKLIST.md)
  - ежедневные operational checks и troubleshooting
- [ARCHITECTURE_OVERVIEW.md](docs/ARCHITECTURE_OVERVIEW.md)
  - технический обзор архитектуры платформы
- [TESTING_GUIDE.md](docs/TESTING_GUIDE.md)
  - подробный guide по тестам, migration workflow и запуску environments
- [USER_JOURNEYS.md](docs/USER_JOURNEYS.md)
  - типовые пользовательские сценарии от compose до живого environment
- [docs/status.md](docs/status.md)
  - фактический task-by-task progress и deviations history

## Основные директории

- [platform](platform)
  - ArgoCD, Crossplane, Backstage, Authentik, NetworkPolicy manifests
- [cli](cli)
  - migration/export/validate/legacy lifecycle CLI
- [templates](templates)
  - Backstage scaffolder templates
- [catalog](catalog)
  - Backstage catalog registration
- [environments](environments)
  - GitOps-managed Environment manifests
- [tests](tests)
  - acceptance, security и load tests

## Canonical Validation Commands

Acceptance:

```bash
go test ./tests/acceptance/... -run TestLifecycle -v
go test ./tests/acceptance/... -run TestComponents -v
go test ./tests/acceptance/... -run TestGitOps -v
go test ./tests/acceptance/... -run TestMigration -v
```

Security:

```bash
go test ./tests/security/... -v
/tmp/kubescape scan platform/
```

Load:

```bash
KUBECONFIG=/root/codex/kubeconfig_6144665 LOAD_TEST_COUNT=50 go test ./tests/load/... -run TestConcurrency -v -timeout 90m
```

## Operational Note

Для live validation и диагностики рекомендуется всегда явно задавать:

```bash
export KUBECONFIG=/root/codex/kubeconfig_6144665
```

Иначе часть `kubectl`-основанных проверок может давать ложные пустые результаты.
