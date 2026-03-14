# New Component Type PR Template

## Назначение

Этот документ — готовый шаблон PR для случая, когда в платформу добавляется
новый supported component type.

Его цель:
- не забыть ни один слой изменений
- упростить review
- сделать PR одинаковыми по структуре

Использовать вместе с:
- [ADDING_NEW_PLATFORM_COMPONENT_TYPE.md](ADDING_NEW_PLATFORM_COMPONENT_TYPE.md)

## Template

Скопируй блок ниже в описание PR и заполни по факту.

```md
## Summary

Adds new platform component type: `<type>`

This PR introduces end-to-end support for `<type>` in the IDP platform:
- provisioning layer
- Environment composition mapping
- CLI conversion / validation updates (if applicable)
- tests
- docs

## Why

Why a new type is needed:
- `<reason 1>`
- `<reason 2>`

Why existing types are insufficient:
- `webapp` is insufficient because `<reason>`
- `postgresql` / `redis` are not applicable because `<reason>`

## User-Facing Behavior

Users can now define:

```yaml
spec:
  components:
    - name: <component-name>
      type: <type>
      enabled: true
```

Expected behavior:
- `<resource/lifecycle behavior>`
- `<enabled/disabled behavior>`
- `<network/db/config behavior>`

## Implementation

### Provisioning

Added:
- `platform/crossplane/<type>/xrd.yaml`
- `platform/crossplane/<type>/composition.yaml`

Updated:
- `platform/crossplane/environment/composition.yaml`

Notes:
- `<provider/function/runtime notes>`

### CLI

Updated:
- `cli/pkg/converter/converter.go` `<if changed>`
- `cli/pkg/validator/validator.go` `<if changed>`

Notes:
- `<mapping heuristics or validation contract>`

### Tests

Added/updated:
- `<unit tests>`
- `<acceptance tests>`
- `<migration tests if applicable>`

### Docs

Updated:
- `<list docs>`

## Validation

### Offline / Static

```bash
kubectl apply --dry-run=client -f platform/crossplane/
```

Result:
- `<pass/fail + notes>`

### Tests

```bash
go test ./cli/... -v
go test ./tests/acceptance/... -run <TestName> -v
```

Result:
- `<pass/fail + notes>`

### Live Smoke

Smoke manifest:

```yaml
apiVersion: idp.platform.io/v1alpha1
kind: Environment
metadata:
  name: <smoke-name>
spec:
  owner: group:default/platform
  team: <team>
  components:
    - name: <component-name>
      type: <type>
      enabled: true
```

Live checks:
- generated Argo Application created
- `Environment` claim reached `Ready=True`
- namespace created
- expected runtime resources created
- disable/delete cleanup path verified

## Risks

- `<risk 1>`
- `<risk 2>`

## Review Checklist

- [ ] New XRD added
- [ ] New Composition added
- [ ] Environment Composition maps new type
- [ ] `enabled: false` path works
- [ ] Runtime resources create successfully
- [ ] Cleanup path works
- [ ] Converter updated if migration should support the type
- [ ] Validator updated if type-specific fields exist
- [ ] Acceptance coverage added
- [ ] Docs updated
- [ ] Live smoke validated
```

## Reviewer Checklist

Reviewer should confirm:
- новый type действительно нужен, а не покрывается существующим
- provisioning shape соответствует runtime semantics
- `Environment` Composition не ломает existing types
- нет скрытых новых provider/function dependencies без wiring
- migration heuristics explainable и не overly aggressive
- есть хотя бы один live smoke path

## Minimal PR Expectations

PR не должен считаться готовым, если:
- type только добавлен в converter, но не в provisioning
- type только добавлен в provisioning, но не протестирован live
- docs не обновлены
- disable/delete path не проверен

## Related Docs

- [ADDING_NEW_PLATFORM_COMPONENT_TYPE.md](ADDING_NEW_PLATFORM_COMPONENT_TYPE.md)
- [TESTING_GUIDE.md](TESTING_GUIDE.md)
- [USER_JOURNEYS.md](USER_JOURNEYS.md)
