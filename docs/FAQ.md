# FAQ

## Назначение

Этот документ отвечает на короткие практические вопросы по эксплуатации и использованию платформы.

## See Also

- [INDEX.md](../INDEX.md)
- [TESTING_GUIDE.md](TESTING_GUIDE.md)
- [USER_JOURNEYS.md](USER_JOURNEYS.md)
- [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)
- [HANDOFF.md](../HANDOFF.md)

## Как попасть в Backstage из браузера?

Добавить в `/etc/hosts`:

```
194.58.110.23 backstage.idp.local
194.58.110.23 authentik-server.authentik.svc.cluster.local
```

Затем открыть:

```
http://backstage.idp.local:30007
```

Backstage — NodePort `30007`, Authentik — NodePort `30009`. Публичный IP кластера — `194.58.110.23`.

Обе записи нужны: при логине браузер редиректится на Authentik для OIDC-авторизации.

## Какой kubeconfig использовать?

Использовать:

```bash
export KUBECONFIG=/root/codex/kubeconfig_6144665
```

Это canonical kubeconfig для live validation в этом репозитории.

## Почему `kubectl` иногда показывает пустой результат, хотя environment должен быть?

Чаще всего причины две:
- не задан правильный `KUBECONFIG`
- Kubernetes API отвечает нестабильно

Сначала проверь:

```bash
echo "$KUBECONFIG"
kubectl get nodes
kubectl get --raw=/readyz
```

## Какое имя у generated Argo Application?

Шаблон:

```text
<team>-<env>
```

Пример:
- manifest `environments/platform/demo.yaml`
- generated app `platform-demo`

## Какое имя у namespace?

Шаблон:

```text
env-<team>-<env>
```

Пример:
- team `platform`
- env `demo`
- namespace `env-platform-demo`

## Что считается главным источником истины?

Git.

Платформа ожидает, что desired state живёт в:
- `environments/`
- `platform/`

Ручной `kubectl apply` не является основным способом управления.

## Можно ли править workload Deployment напрямую через `kubectl`?

Можно как debug-операцию, но это не правильный путь управления.

Нужно помнить:
- Argo владеет manifest/claim layer
- Crossplane владеет runtime provisioning layer

Поэтому прямые правки workload resources не должны считаться canonical способом изменения environment.

## Как создать новый test environment?

Есть два нормальных варианта:

1. вручную через Git manifest в `environments/<team>/<env>.yaml`
2. через Backstage scaffolder template

Подробный workflow:
- [TESTING_GUIDE.md](TESTING_GUIDE.md)
- [USER_JOURNEYS.md](USER_JOURNEYS.md)

## Как удалить environment?

Удалить manifest из Git:

```bash
git rm environments/<team>/<env>.yaml
git commit -m "Delete environment <team>/<env>"
git push origin feature/phase-1-foundation
```

Потом проверить, что:
- generated `Application` удалён
- namespace удалён

## Как временно выключить один сервис?

В manifest поставить:

```yaml
enabled: false
```

Это правильный способ убрать component, не удаляя весь environment.

## Какие component types поддерживаются сейчас?

На текущем этапе:
- `webapp`
- `postgresql`
- `redis`

Если сервис не подходит ни под один тип:
- сначала оставить его `enabled: false`
- потом решать, нужен ли новый component type

## Что делать с unknown service после `idp migrate`?

Не пытаться сразу насильно включить его.

Правильный путь:
1. оставить `enabled: false`
2. посмотреть TODO/manual review warning
3. понять, можно ли выразить сервис через existing type
4. если нельзя, спроектировать новый platform type

## Как мигрировать legacy приложение из `docker-compose`?

Базовый путь:

```bash
./cli/idp migrate --from ./docker-compose.yml --output ./environment.yaml
./cli/idp validate --file ./environment.yaml
```

Потом:
- review output
- commit в `environments/`
- проверить generated app / claim / namespace

Подробно:
- [TESTING_GUIDE.md](TESTING_GUIDE.md)
- [USER_JOURNEYS.md](USER_JOURNEYS.md)

## Как зарегистрировать legacy environment в tracker?

```bash
export IDP_TRACKER_DB=/path/to/tracker.db
./cli/idp register-legacy --name <name> --team <team> --compose-path ./docker-compose.yml
```

## Почему `complete-migration` может не работать?

Потому что environment ещё не переведён в `validated`.

`complete-migration` работает только для записей со статусом:

```text
validated
```

## Где искать deprecation report?

В директории, заданной через:

```bash
export IDP_REPORT_DIR=/path/to/reports
```

Файл будет вида:

```text
deprecation-report-YYYY-MM-DD.md
```

## Что делать, если manifest уже в Git, а environment не появился?

Идти по цепочке:

1. проверить, что файл реально есть в remote branch
2. проверить generated `Application`
3. проверить `Environment` claim
4. проверить namespace
5. проверить runtime resources

Полезный документ:
- [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)

## Что запускать как минимальный smoke test?

```bash
go test ./tests/security/... -v
go test ./tests/acceptance/... -run TestLifecycle -v
```

Если это зелёное, система обычно здорова на базовом уровне.

## Когда запускать full load test?

Когда менялось что-то, влияющее на:
- ArgoCD reconcile path
- Crossplane provisioning performance
- GitOps branch update pattern
- cluster capacity

Canonical команда:

```bash
KUBECONFIG=/root/codex/kubeconfig_6144665 LOAD_TEST_COUNT=50 go test ./tests/load/... -run TestConcurrency -v -timeout 90m
```

## Почему acceptance/load tests делают Git commits?

Потому что платформа GitOps-driven, и честная validation должна проходить через реальный desired-state path.

## Это безопасно запускать на рабочей branch?

Технически да, но operationally лучше запускать там, где допустимы служебные test commits.

Особенно это важно для:
- acceptance suites
- load tests

## Что проверяет security suite?

1. отсутствие literal credentials в `environments/`
2. наличие baseline network policies в live env namespaces

Команда:

```bash
go test ./tests/security/... -v
```

## Почему Kubescape запускается только на `platform/`?

Потому что для repo-hosted platform manifests это meaningful scan target.

Для `environments/` основной security control здесь другой:
- literal credential audit через Go test

## Где читать подробнее?

Если нужен:
- обзор: [RELEASE_SUMMARY.md](../RELEASE_SUMMARY.md)
- handoff: [HANDOFF.md](../HANDOFF.md)
- operations: [OPERATIONS_CHECKLIST.md](../OPERATIONS_CHECKLIST.md)
- подробный usage guide: [TESTING_GUIDE.md](TESTING_GUIDE.md)
- user scenarios: [USER_JOURNEYS.md](USER_JOURNEYS.md)
