# Implementation Status: IDP Test Environment Platform

## Current Position
- Phase: phase-2.7
- Task: task-2.7.4
- Status: IN_PROGRESS

## Progress

### Phase 1: Project Foundation
| Task     | Status        | Notes |
|----------|---------------|-------|
| task-1.1 | COMPLETE | Scaffolded required top-level directories, root README.md, .gitignore; initialised local git metadata to run validation. |
| task-1.2 | COMPLETE | Authored Environment CRD, added schema bounds for CEL cost budget, validated with kubectl dry-run, confirmed duplicate component names are rejected by the API, then removed the temporary CRD from the cluster. |
| task-1.3 | COMPLETE | Initialised Go CLI module, added command entrypoints and shared CLI harness, defined required pkg interfaces/types, validated compilation with Go 1.22 toolchain in /tmp. |
| task-1.4 | COMPLETE | Added five docker-compose fixtures and two expected Environment manifests; validated fixture parsing with compose-go and schema-validated expected CRs against the Environment CRD. |
| task-1.5 | COMPLETE | Created CI workflow skeletons, fixed CRD validation to run against an ephemeral kind cluster, and verified all CI jobs succeeded on GitHub Actions for branch feature/phase-1-foundation. |

### Phase 2: Platform Infrastructure
| Task     | Status        | Notes |
|----------|---------------|-------|
| task-2.1 | COMPLETE | Added XPostgresqlInstance XRD and a pipeline Composition using go-templating/auto-ready; validated with crossplane beta validate and render for enabled=true/false including correct Secret, Service, PVC, and Deployment naming. |
| task-2.2 | COMPLETE | Added XRedisInstance XRD and a pipeline Composition; validated with crossplane beta validate and render for enabled=true/false including correct Secret and Service naming plus empty disabled output. |
| task-2.3 | COMPLETE | Added XWebappInstance XRD and pipeline Composition; validated replicas=3, replicas=0, enabled=false, imageTag length enforcement, and nested postgresql composite rendering. |
| task-2.4 | COMPLETE | Added cluster-defaults EnvironmentConfig, switched compositions to EnvironmentConfig-driven defaults, validated exact kubectl dry-run for the manifest, and confirmed default injection in postgresql and webapp renders. |
| task-2.5 | COMPLETE | Added ArgoCD Applications and ApplicationSet manifests, bootstrapped ArgoCD on the cluster, verified platform apps/AppSet creation, generated exactly one temporary test-smoke Application from Git, and confirmed it was pruned after manifest deletion. |
| task-2.6 | COMPLETE | Added baseline namespace isolation policies and validated live behavior: intra-namespace HTTP succeeded, cross-namespace HTTP timed out, and DNS lookup to kube-system succeeded. |

### Phase 2.7: Environment Reconciliation
| Task       | Status        | Notes |
|------------|---------------|-------|
| task-2.7.1 | COMPLETE | Replaced the plain CRD with a Crossplane XRD + namespaced claim in `platform/crds/environment.yaml`, validated it offline with `crank beta validate`, validated `kubectl apply --dry-run=client`, applied the XRD to the cluster, and confirmed a duplicate-component claim is rejected by CEL with `Component names must be unique within an environment`. |
| task-2.7.2 | COMPLETE | Added `platform/crossplane/environment/composition.yaml`; validated with `crank beta validate` and rendered a fixture `XEnvironment` to confirm one derived namespace, all three baseline NetworkPolicies, `XWebappInstance`, `XPostgresqlInstance`, and omission of the disabled `redis` component from the desired child composite set. |
| task-2.7.3 | COMPLETE | Added `platform/crossplane/provider-kubernetes.yaml`, installed `provider-kubernetes` v0.18.0, waited for the provider CRD registration race to settle, re-ran `kubectl apply --dry-run=client -f platform/crossplane/`, and confirmed `provider-kubernetes` is Healthy plus `kubernetes-provider` exists as a cluster `ProviderConfig` using `InjectedIdentity`. |
| task-2.7.4 | IN_PROGRESS | User approved the scope extension to remediate the legacy component XRD artifacts from tasks 2.1–2.3 before wiring ArgoCD to GitOps-deploy `platform/crossplane/`. |
| task-2.7.5 | NOT_STARTED | |

### Phase 3: CLI Migration Tool
| Task     | Status        | Notes |
|----------|---------------|-------|
| task-3.1 | COMPLETE | Implemented ComposeConverter with compose-go loading, component mapping, truncation-based namespace generation, manual-review reporting, and fixture coverage at 82.5%. |
| task-3.2 | COMPLETE | Implemented read-only LegacyExporter with compose/runtime fallback, secret redaction, partial-result handling, and tests at 95.2% coverage with no write-path behavior. |
| task-3.3 | COMPLETE | Implemented CRD-backed EnvironmentValidator using k8s apiextensions schema validation plus duplicate-name, replicas, OCI tag, and config override checks; tests pass at 83.2% coverage. |
| task-3.4 | COMPLETE | Wired Cobra-based migrate/export/validate commands, added root cmd/idp entrypoint, verified static build plus migrate/validate happy-path and missing-file failure handling. |
| task-3.5 | COMPLETE | Added pure-Go SQLite tracker, lifecycle commands for register/complete/deprecate, validated the full register→validated→complete→deprecate flow including report generation and deprecated flag persistence. |
| task-3.6 | COMPLETE | Added integration_test.go under build tag integration covering migrate→validate, 10-service mapping accuracy (9/9 mapped), retention expiry helper logic, and the register→complete→deprecate lifecycle. |

### Phase 4: Backstage Portal
| Task     | Status        | Notes |
|----------|---------------|-------|
| task-4.1 | COMPLETE | Added Backstage OIDC config plus `auth.session.secret`, pushed a custom Backstage image with the OIDC provider module, created the Authentik provider/application and `backstage-secrets`, verified `GET /api/auth/oidc/start` returns a `302` redirect to Authentik, and validated test-user claims: `dev-alice` => `role=developer, team_id=alpha`, `platform-bob` => `role=platform_engineer, team_id=null`. |
| task-4.2 | COMPLETE | Added static GitHub-backed catalog locations with `integrations.github` token wiring, pinned the initial catalog target to commit `1a7c2a4` to avoid branch-name slash parsing, corrected Kubernetes `caData` to the base64-encoded cluster CA, validated RBAC dry-run, confirmed authenticated catalog API returns 4 entities including `component:default/backstage-portal`, and confirmed authenticated Kubernetes workload queries return live pod/service/deployment data for that component. |
| task-4.3 | COMPLETE | Added the `new-environment` scaffolder template plus skeleton files, packaged the catalog and templates into the Backstage image, registered the template from a local file-backed catalog, added a custom `github:repo:upsert` scaffolder action for updating the existing GitOps branch, validated a full live task run (`e9507faa-e4a7-48b1-a0b3-a3cf80e209e3`) that committed branch head `0c95e8ead5c0ebda2b8bf34dc509109133e32915`, generated `environments/platform/smoke-template-0314h.yaml` and `catalog/environments/platform-smoke-template-0314h.yaml`, and registered `component:default/platform-smoke-template-0314h` in the catalog. |
| task-4.4 | COMPLETE | Added `packages/backend/src/plugins/migrationDashboard.ts`, implemented and loaded a runtime Backstage `migration` backend plugin through the custom image path, exposed `GET /api/migration/status`, and validated live aggregation from SQLite tracker data in pod `backstage-6565c6cfd9-g6wkp`: 3 records with 2 completed returned `percentage=66`, then replacing the same tracker DB with a 4th registration immediately returned `total=4` and `percentage=50` with no restart. |
| task-4.5 | COMPLETE | Added `packages/app/src/components/MigrationDashboard/index.tsx` and exposed a live `/migration` page through Backstage runtime wiring on `rootHttpRouter`; validated live that the page renders the required Team/Total/Completed/Validated/In Progress/Blocked/% table, contains a 60s refresh loop, and auto-updates without reload: a jsdom execution of the page script saw initial rows `alpha 3 2 0 0 0 66%`, then after replacing the tracker DB it refreshed to `alpha 4 4 0 0 0 100%` and made the `Gate PASSED` banner visible (`display: block`). |
| task-4.6 | COMPLETE | Added the three migration runbooks plus `platform/backstage/app-config.techdocs.yaml`, packaged runbooks into the Backstage image, exposed a live `/docs` runbook portal through runtime wiring, and validated locally on `http://localhost:7007/docs` with Postgres-backed Backstage startup: landing page listed all 3 application types, each runbook page rendered all 5 UC-018 steps, and `/docs/runbooks/unknown-type` returned `200` with `Runbook missing — contact platform engineer`. |

### Phase 5: Acceptance Testing & Hardening
| Task     | Status        | Notes |
|----------|---------------|-------|
| task-5.1 | NOT_STARTED | Deferred until phase-2.7 is implemented because lifecycle acceptance tests require a real Environment -> child composite provisioning path. |
| task-5.2 | NOT_STARTED | |
| task-5.3 | NOT_STARTED | |
| task-5.4 | NOT_STARTED | |
| task-5.5 | NOT_STARTED | |
| task-5.6 | NOT_STARTED | |

## Checkpoints
| Phase   | Status      | Approved |
|---------|-------------|----------|
| phase-1 | REACHED | |
| phase-2 | REACHED | |
| phase-2.7 | NOT_REACHED | |
| phase-3 | REACHED | |
| phase-4 | REACHED | |
| phase-5 | NOT_REACHED | |

## Blockers

## Deviations
- task-4.3: User approved extending scope to edit `catalog/all-components.yaml` so the new scaffolder template can be registered in the static Backstage catalog.
- task-4.3: User approved implementing a custom scaffolder backend action because the stock `github:repo:push` action could not safely update the existing populated GitOps branch.
- task-4.4: User approved extending scope to runtime wiring changes because the repository has no Backstage source workspace; the backend plugin is authored at `packages/backend/src/plugins/migrationDashboard.ts` for spec compliance and loaded in the live portal through the prebuilt-image patch path in `platform/backstage/Dockerfile`.
- task-4.5: User approved extending scope to runtime/build wiring changes because the repository has no Backstage app source workspace; the frontend source artifact is authored at `packages/app/src/components/MigrationDashboard/index.tsx`, while the live `/migration` route is served through `rootHttpRouter` in the prebuilt-image runtime path.
- task-4.6: Implemented the required TechDocs runbooks under `docs/runbooks/`, but because the repository has no Backstage frontend source workspace or TechDocs content pipeline, the live `/docs` experience is served through a custom runtime module packaged into the prebuilt image while `platform/backstage/app-config.techdocs.yaml` and `helm-values.yaml` carry the TechDocs configuration.
- phase-2.7: Plan revised after phase-4 approval to add the previously unscheduled Environment claim/XRD, Environment Composition, Crossplane provider wiring, Crossplane Argo multi-source deployment, and end-to-end reconciliation smoke validation required before phase-5 acceptance tests can be meaningful.
