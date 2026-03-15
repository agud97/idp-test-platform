# Technical Constraints — IDP Test Environment Platform

**Version:** 1.0
**Date:** 2026-03-14
**Audience:** AI coding agent, software engineers
**Linked specs:** [vision.md](vision.md) · [requirements.md](requirements.md) · [entity_model.md](entity_model.md) · [use_cases/](use_cases/) · [acceptance_criteria.md](acceptance_criteria.md)

> **How to read this document**
> - **MUST** — mandatory. Implementation is invalid without it.
> - **SHOULD** — strong preference. Deviation requires explicit justification.
> - **MUST NOT** — explicit prohibition. Any violation is a defect.

---

## 1. Project Structure

### 1.1 Repository Layout

**MUST** organise the repository with the following top-level directories:

```
idp-platform/
├── platform/               # Platform infrastructure managed by ArgoCD
│   ├── crds/               # Environment claim/XRD YAMLs
│   ├── crossplane/         # XRDs and Compositions per component type
│   │   └── <component-type>/
│   │       ├── composition.yaml
│   │       └── xrd.yaml
│   ├── argocd/             # ArgoCD Applications and ApplicationSets
│   └── networkpolicies/    # Default namespace isolation policies
├── environments/           # User environment manifests (watched by ArgoCD)
│   └── <team>/
│       └── <env-name>.yaml
├── catalog/                # Backstage catalog-info.yaml entries
├── templates/              # Backstage scaffolder templates
│   └── <template-name>/
│       ├── template.yaml
│       └── skeleton/
│           └── environments/${{ values.team }}/${{ values.env_name }}.yaml
├── cli/                    # Migration CLI tool (Go)
│   ├── cmd/
│   │   ├── migrate/
│   │   ├── export/
│   │   └── validate/
│   ├── pkg/
│   │   ├── converter/
│   │   ├── exporter/
│   │   └── validator/
│   └── internal/
└── docs/                   # All specification and architecture documents
```

**MUST** place all user-created environment manifests under `environments/<team>/`. ArgoCD ApplicationSet **MUST** watch exactly this path with `recurse: true`.

**MUST NOT** place environment manifests in `platform/`. The `platform/` directory is reserved for infrastructure that only platform engineers modify.

**MUST** store each Crossplane Composition in its own subdirectory named after the component type it implements (e.g., `platform/crossplane/postgresql/`, `platform/crossplane/redis/`).

**SHOULD** keep each Backstage scaffolder template in its own subdirectory under `templates/`. One directory = one template.

### 1.2 CLI Tool Structure (Go)

**MUST** implement the migration CLI as a Go module with the module path `github.com/<org>/idp-platform/cli`.

**MUST** separate command entry points (`cmd/`) from reusable business logic (`pkg/`). The `cmd/` packages **MUST NOT** contain business logic; they **MUST** only parse flags, call `pkg/` functions, and format output.

**MUST** place all internal helpers that are not part of the public API under `internal/`. Packages in `internal/` **MUST NOT** be imported by external modules.

**MUST** define a dedicated package for each CLI sub-command: `pkg/converter`, `pkg/exporter`, `pkg/validator`.

---

## 2. Component Design

### 2.1 Environment Claim / XRD

**MUST** define the Environment as a Crossplane-backed Kubernetes Custom Resource claim. The backing definition **MUST** be a `CompositeResourceDefinition` that exposes a namespaced claim with:
- `group: idp.platform.io`
- `kind: Environment`
- API version: `v1alpha1` initially, promoting to `v1` when the schema is stable
- OpenAPI v3 schema with `x-kubernetes-validations` (CEL rules) enforcing business rules

**MUST** include the following top-level fields in the Environment spec:

```yaml
spec:
  owner: string           # required
  team: string            # required
  components:
    - name: string        # required, unique within spec
      type: string        # required, must match a registered ComponentType
      enabled: boolean    # required
      imageTag: string    # optional
      replicas: integer   # optional, 0–50
      configOverrides: {}  # optional, map[string]string
```

**MUST** add a CEL validation rule that enforces component `name` uniqueness within a single Environment spec:
```yaml
x-kubernetes-validations:
  - rule: "self.components.all(c, self.components.filter(x, x.name == c.name).size() == 1)"
    message: "Component names must be unique within an environment"
```

**MUST** define the Environment `status` subresource with fields: `phase` (Synced | Progressing | Degraded | Deleting), `lastSyncedCommit`, `message`.

**MUST** implement a dedicated Environment Composition that maps one committed `kind: Environment` claim into:
- one workload namespace following the environment naming convention
- the three baseline NetworkPolicies in that namespace
- one child composite per `spec.components[]` entry (`XWebappInstance`, `XPostgresqlInstance`, `XRedisInstance` as applicable)

**MUST NOT** require developers to commit lower-level `XWebappInstance`, `XPostgresqlInstance`, or `XRedisInstance` resources directly. The Environment claim is the only user-facing provisioning API.

**MUST NOT** store database credentials, passwords, or any secrets in the Environment spec or status. See constraint 2.4.

### 2.2 Crossplane Compositions

**MUST** define one `CompositeResourceDefinition` (XRD) and one `Composition` per component type.

**MUST** implement the `enabled` flag using a Crossplane Composition `patch` that conditionally includes or excludes managed resources based on the field value:

- `enabled: true` → Crossplane creates/maintains all managed resources (Deployment, Service, ConfigMap, DatabaseInstance).
- `enabled: false` → Crossplane sets `deletionPolicy: Delete` on all managed resources for that component and removes them from the desired state, causing the provider to delete them. All resources (Deployment, Service, ConfigMap, DatabaseInstance, credentials Secret) **MUST** be fully deleted within 3 minutes. (ref: UC-003, AC-005)

> Note: `replicas: 0` is a separate concept from `enabled: false`. Setting `replicas: 0` scales the Deployment to zero pods but keeps all resources. Setting `enabled: false` removes all resources entirely.

**MUST** ensure Compositions are idempotent: if `enabled: false` is applied to a component that was never enabled, no resources are created and no error is raised.

**MUST** implement database provisioning as a nested Composite Resource (`XDatabaseInstance`) referenced from the component Composition. The database Composition **MUST NOT** be called directly from environment manifests.

**SHOULD** use Crossplane `EnvironmentConfigs` to inject cluster-wide defaults (default storage class, default image registry) into Compositions rather than hardcoding values.

### 2.3 ArgoCD ApplicationSet

**MUST** use an `ApplicationSet` with a `git` generator (**files** strategy, NOT directories) to create one ArgoCD `Application` per environment manifest file. Environments are stored as individual YAML files, not as directories — using the `directories` generator would create one Application per team directory, which is incorrect.

```yaml
generators:
  - git:
      repoURL: https://github.com/<org>/idp-platform
      revision: HEAD
      files:
        - path: environments/**/*.yaml
```

**MUST** configure the ApplicationSet template with:
```yaml
syncPolicy:
  automated:
    prune: true
    selfHeal: true
  syncOptions:
    - CreateNamespace=true
    - RespectIgnoreDifferences=true
```

**MUST NOT** create individual ArgoCD `Application` resources manually per environment. All Applications **MUST** be generated by the ApplicationSet. (ref: FR-013, AC-037–040)

**MUST** apply committed Environment claims into a stable control namespace used for claims and reconciliation metadata. The ArgoCD Application destination namespace **MUST NOT** be the workload namespace, because the workload namespace is created by the Environment Composition itself.

**MUST** configure `ignoreDifferences` for CRD status fields to prevent reconciliation loops on Crossplane-managed resources:
```yaml
ignoreDifferences:
  - group: apiextensions.crossplane.io
    kind: CompositeResource
    jsonPointers:
      - /status
```

### 2.4 Secret Management

**MUST** never write database credentials, API keys, or any secrets to Git. (ref: AC-018, BR-002 in UC-016)

**MUST** generate database credentials at provisioning time inside the Crossplane Composition using a `ProviderConfig` or an `InitProvider`. Credentials **MUST** be written to a Kubernetes `Secret` in the environment namespace.

**MUST** name the credentials Secret following the pattern: `<component-name>-db-credentials`. The component Deployment **MUST** reference it via `envFrom.secretRef`, never via hardcoded `env.value`.

**MUST NOT** use `stringData` in Secret manifests committed to Git, even with placeholder values.

### 2.5 CLI Tool — Interfaces

**MUST** define the following Go interfaces in `pkg/converter`, `pkg/exporter`, and `pkg/validator`:

```go
// pkg/converter
type ComposeConverter interface {
    Convert(input io.Reader) (*EnvironmentManifest, *ConversionReport, error)
}

// pkg/exporter
type LegacyExporter interface {
    Export(envName string, opts ExportOptions) (*LegacyConfig, *ExportSummary, error)
}

// pkg/validator
type EnvironmentValidator interface {
    Validate(manifest *EnvironmentManifest) []ValidationError
}
```

**MUST** accept `io.Reader` / `io.Writer` as parameters instead of file paths in all `pkg/` functions. File I/O **MUST** be performed only in `cmd/` layer.

**MUST NOT** use global state or package-level variables in `pkg/` packages. All dependencies **MUST** be injected via constructor functions.

### 2.6 NetworkPolicy Design

**MUST** apply the following two NetworkPolicy resources to every environment namespace immediately after namespace creation:

1. **`deny-cross-namespace-ingress`** — denies all ingress from pods outside the namespace
2. **`allow-intra-namespace`** — allows all traffic between pods within the same namespace

**MUST** apply a third policy `allow-dns-egress` that permits UDP/TCP egress on port 53 to `kube-system` namespace (cluster DNS).

**MUST NOT** apply a blanket `allow-all` NetworkPolicy at any scope. All exceptions beyond the three baseline policies must be explicitly declared. (ref: NFR-005, AC-041–042)

The three baseline NetworkPolicies **MUST** be created by the Environment Composition in each derived workload namespace. Static manifests under `platform/networkpolicies/` are templates and source artifacts, not standalone shared policies.

---

## 3. Technology Decisions

### 3.1 Core Platform Tools

**MUST** use the following tools with no substitutions (ref: C-001, C-002, C-003):

| Tool | Minimum version | Purpose |
|---|---|---|
| Kubernetes | 1.26 | Runtime platform |
| ArgoCD | 2.9 | GitOps reconciliation |
| Crossplane | 1.14 | Composition engine |
| Backstage | 1.24 | Developer portal |

**MUST NOT** introduce any secondary CD tool (Flux, Helm Operator, Kustomize controller) for environment management. ArgoCD is the sole reconciliation engine. (ref: C-001)

**MUST NOT** use cloud-provider managed database services (AWS RDS, GCP CloudSQL, Azure Database). All databases **MUST** run as in-cluster workloads. (ref: C-009)

### 3.2 CLI Tool — Language & Libraries

**MUST** implement the migration CLI in Go 1.22 or later.

**MUST** use the following libraries:

| Library | Purpose |
|---|---|
| `sigs.k8s.io/yaml` | YAML parsing and serialisation |
| `github.com/compose-spec/compose-go` | docker-compose file parsing |
| `github.com/spf13/cobra` | CLI command structure |
| `k8s.io/apiextensions-apiserver` | CRD schema validation |
| `github.com/stretchr/testify` | Test assertions |

**MUST NOT** use `gopkg.in/yaml.v2`. Use `sigs.k8s.io/yaml` exclusively for Kubernetes-compatible YAML handling.

**MUST NOT** shell out to `kubectl`, `docker`, or `docker-compose` binaries from within the CLI tool. All operations **MUST** use Go library calls.

**MUST** support docker-compose files conforming to the **Docker Compose Specification** (compose-spec v1.0+), which covers `version: "3.x"` and unversioned files. Files using `version: "2.x"` syntax **SHOULD** be supported via `compose-go`'s backward-compat loader. Unsupported `version: "2.x"` directives **MUST** be listed in the conversion report rather than causing a hard failure.

### 3.3 Backstage Configuration

**MUST** configure Backstage with the following plugins enabled:
- `@backstage/plugin-catalog` — software catalog
- `@backstage/plugin-scaffolder` — environment creation templates
- `@backstage/plugin-kubernetes` — live status from ArgoCD/cluster
- `@backstage/plugin-techdocs` — migration runbooks

**MUST** register the environment `catalog-info.yaml` entries using static `catalog.locations` in `app-config.yaml` pointing to the `catalog/` directory in Git. **MUST NOT** rely on GitHub catalog discovery (`@backstage/plugin-catalog-backend-module-github`) as it is not included in the default Backstage image. (ref: memory — Backstage known issue)

**MUST** use a dedicated Kubernetes ServiceAccount `backstage-k8s-reader` with a ClusterRole limited to `get`, `list`, `watch` on pods, deployments, replicasets, and services. **MUST NOT** grant cluster-admin to the Backstage service account.

**MUST** authenticate Backstage users via the existing **Authentik** OAuth2/OIDC provider on the cluster. Configure Backstage `auth.providers.oidc` with `metadataUrl: http://authentik-server.authentik.svc.cluster.local/application/o/backstage/.well-known/openid-configuration` (port **80**, not 9000). Guest authentication **MUST** be disabled in production (`providers.guest: null`). (ref: NFR-006, AC-025, AC-026)

**MUST** set `prompt: select_account` in the OIDC provider config. The Backstage default (`prompt: none`) causes Authentik to return `login_required` immediately for unauthenticated users; `prompt: login` causes an infinite re-authentication loop in Authentik. `select_account` is the only value that allows interactive login without looping.

**MUST** account for the prebuilt Backstage image limitation: the sign-in page UI is compiled into the frontend JS bundle and is **not configurable via `app.signInPage` in app-config**. To replace the hardcoded guest Component with OIDC:
- Patch `module-backstage.*.js` in the `Dockerfile` to replace the guest `Component` with an OIDC popup and the `B` loader with a session-check via `/api/auth/oidc/refresh?env=production`.
- Rename the patched file (e.g., `module-backstage.oidcpatch.js`) to bust the 2-week browser cache (`Cache-Control: public, max-age=1209600`).
- Patch **both** `/app/packages/app/dist/index.html` **and** `/app/packages/app/dist/index.html.tmpl` to reference the renamed file. The app-backend serves HTML from `index.html.tmpl` (not `index.html`) — patching only `index.html` has no effect.
- The identity object passed to `onSignInSuccess` **MUST** implement the full `IdentityApi` interface: `getBackstageIdentity()`, `getProfileInfo()` (NOT `getProfile()`), `getCredentials()` returning `{token: string}`, and `signOut()`. Using `getProfile` instead of `getProfileInfo` causes `TypeError: this.config.identityApi.getProfileInfo is not a function` after sign-in.

**MUST** seed the Backstage catalog with a `kind: User` entity for every real user who will log in, with `spec.profile.email` matching the email in Authentik. The default sign-in resolver (`emailMatchingUserEntityProfileEmail`) looks up the user in the catalog by email; if the entity is absent, sign-in fails. At minimum, the admin user `akadmin` (email `root@example.com`) **MUST** be present in `catalog/all-components.yaml`. The catalog location **MUST** include `User` in its `rules.allow` list.

**MUST** implement the migration tracking dashboard as a **custom Backstage plugin** (`@internal/plugin-migration-dashboard`) that:
- Renders at route `/migration`
- Fetches data from a backend API endpoint `GET /api/migration/status` that returns per-team aggregates derived from `LEGACY_ENVIRONMENT` records
- Shows a table: team name, total environments, migrated count, validated count, blocked count, percentage
- Refreshes data every 60 seconds
The backend plugin **MUST** be implemented as a Backstage backend plugin at `packages/backend/src/plugins/migrationDashboard.ts`. (ref: MAJ-003, UC-019, AC-065)

**MUST** treat this repository's Backstage distribution as a prebuilt-image deployment, not a full source monorepo. If `packages/backend/` or `packages/app/` source artifacts are not part of the live build graph, the live behavior **MUST** be wired through the image build/runtime integration under `platform/backstage/` while preserving the specified source-artifact files for traceability and spec compliance.

**MUST** register scaffolder templates through the static Backstage catalog configuration that the running image actually ingests. Placing a template file under `templates/` alone is insufficient; the template **MUST** be reachable through `catalog.locations` or an equivalent file-backed catalog registration path packaged into the image.

**MUST** use `spec.type: service` (not `environment`) for all environment catalog entries. In the prebuilt Backstage image, the Kubernetes tab is only rendered for `spec.type: service` entities. Other types fall into a default layout with no Kubernetes tab. This means the scaffolder template skeleton (`templates/new-environment/skeleton/catalog-info.yaml`) and all catalog entries under `catalog/environments/` MUST use `type: service`.

### 3.3.1 Known Limitation: Dockerfile JS Patching is Technical Debt

The OIDC sign-in currently works via **direct patching of compiled JavaScript in the Dockerfile** (`module-backstage.*.js`). This is a PoC-acceptable workaround with the following known risks:

| Risk | Impact |
|------|--------|
| Patch is tied to specific rspack bundle internals (variable names `B`, `Component`) — these can change on any Backstage version bump | Patch may silently break on Backstage update |
| `IdentityApi` is emulated manually — interface changes in future Backstage versions require manual patch updates | Regression risk on upgrades |
| `app.signInPage: oidc` in app-config has **no effect** in the prebuilt image | Behavior diverges from official Backstage documentation |

**Long-term correct approach:** Build Backstage from source (`npx @backstage/create-app`, configure `OIDCSignInPage` in `packages/app/src/App.tsx`, run `yarn build`). In a source build, `app.signInPage: oidc` works natively via runtime app-config injection, and the `IdentityApi` is provided by `@backstage/plugin-auth-react` without custom patching.

The current Dockerfile patch approach **SHOULD NOT** be used in a production-grade deployment. Migration to a source build is recommended before promoting this platform beyond PoC/demo scope.

### 3.4 Crossplane Provider

**MUST** use `provider-kubernetes` (crossplane-contrib) for creating in-cluster Kubernetes resources (Deployments, Services, ConfigMaps, Secrets) from Compositions.

**MUST** use `provider-helm` only for components that are exclusively distributed as Helm charts and have no viable Kubernetes-native alternative.

**MUST** define a `ProviderConfig` named `kubernetes-provider` through repo-hosted manifests committed under `platform/crossplane/`. The provider installation and ProviderConfig **MUST** be applied by the ArgoCD Crossplane Application, not by imperative commands.

**MUST** configure the ArgoCD Crossplane Application as a multi-source Application: one source for the upstream Crossplane chart and one source for repo-hosted definitions (`platform/crds/` and `platform/crossplane/`).

**MUST NOT** use cloud-provider Crossplane providers (`provider-aws`, `provider-gcp`, `provider-azure`) in the platform stack. (ref: C-009)

### 3.5 Database Component Images

**MUST** use the following default images for managed database components:

| Engine | Default image |
|---|---|
| PostgreSQL | `bitnamilegacy/postgresql:16` |
| Redis | `bitnamilegacy/redis:8.2.1-debian-12-r0` |

**SHOULD** allow the image tag to be overridden per component via `imageTag` in the environment manifest.

**MUST NOT** use `bitnami/postgresql` or `bitnami/redis` from docker.io as they return 403 errors. Use `bitnamilegacy/` prefix. (ref: memory — SSO Stack)

---

## 4. Code Style

### 4.1 Naming Conventions

**MUST** follow these naming rules across all YAML manifests:

| Resource | Convention | Example |
|---|---|---|
| Environment namespace | `env-<team>-<env-name>` (max 63 chars — see truncation rule) | `env-platform-my-app` |
| ArgoCD Application | `<team>-<env-name>` | `platform-my-app` |
| DB credentials Secret | `<component-name>-db-credentials` | `backend-db-credentials` |
| NetworkPolicy | `<scope>-<direction>-<subject>` | `deny-cross-namespace-ingress` |
| Crossplane XRD kind | `X<ComponentType>Instance` | `XPostgresqlInstance` |
| Crossplane Composition | `<component-type>-composition` | `postgresql-composition` |

**MUST** apply the following truncation rule when the derived namespace name would exceed 63 characters: truncate `<env-name>` to fit within the limit and append a 4-character lowercase hex hash of the full untruncated name to preserve uniqueness. Formula: `env-<team[:20]>-<env-name[:N]>-<xxxx>` where N = 63 − 4 (prefix) − len(team[:20]) − 1 (dash) − 1 (dash) − 4 (hash) = 33. Implementation: `shortName = fmt.Sprintf("env-%s-%s-%s", team[:min(20,len(team))], envName[:min(N, len(envName))], hash4(team+"-"+envName))`.

**MUST** follow standard Go naming conventions: `PascalCase` for exported identifiers, `camelCase` for unexported, `SCREAMING_SNAKE_CASE` for constants.

**MUST** name Go interfaces with a single-method suffix pattern where applicable: a single-method interface for converting **MUST** be named `Converter`, not `IConverter` or `ConvertInterface`.

**MUST NOT** prefix Go interfaces with `I` (e.g., `IConverter`). This is a Java convention and is not idiomatic Go.

### 4.2 YAML Style

**MUST** use 2-space indentation in all YAML files.

**MUST** include `metadata.labels` with at minimum:
```yaml
labels:
  app.kubernetes.io/managed-by: crossplane   # for Compositions
  app.kubernetes.io/managed-by: argocd       # for ArgoCD Applications
  idp.platform.io/environment: <env-name>
  idp.platform.io/team: <team>
```

**MUST NOT** use YAML anchors (`&`, `*`) in environment manifests. Templates and Compositions **SHOULD** avoid anchors to maintain readability.

**MUST** add a `# Source: <tool>` comment at the top of generated files (e.g., files produced by the Backstage scaffolder or the migration CLI).

### 4.3 Go Code Patterns

**MUST** return errors as the last return value. **MUST NOT** panic in `pkg/` packages. Panics are only acceptable in `main()` for unrecoverable startup failures.

**MUST** wrap errors with context using `fmt.Errorf("converting service %q: %w", name, err)`.

**MUST** use structured logging (`log/slog`) rather than `fmt.Println` or `log.Printf` for all CLI output that is not user-facing result data.

**MUST NOT** use `init()` functions in any `pkg/` or `internal/` package.

**SHOULD** use the functional options pattern for constructors with more than 3 parameters:
```go
func NewConverter(opts ...ConverterOption) *Converter
```

### 4.4 Anti-Patterns to Avoid

**MUST NOT** hardcode cluster endpoint URLs, namespaces, or image registries as string literals in Go code. Use configuration structs loaded from environment variables or config files.

**MUST NOT** use `kubectl exec` or shell commands inside the CLI tool. (ref: constraint 3.2)

**MUST NOT** commit any file containing real credentials, tokens, or private keys. The `.gitignore` **MUST** exclude `*.env`, `kubeconfig*`, `token-*`, and `*-secret.yaml`.

**MUST NOT** write environment-specific logic (if-statements checking environment name or team name) inside Crossplane Compositions. Compositions **MUST** be generic and data-driven.

**MUST NOT** create a single God-Composition that handles all component types. One component type = one Composition.

---

## 5. Testing Strategy

### 5.1 CLI Unit Tests

**MUST** write unit tests for every public function in `pkg/converter`, `pkg/exporter`, and `pkg/validator`.

**MUST** achieve minimum 80 % line coverage on `pkg/` packages, measured with `go test -cover`.

**MUST** test the converter against fixture files in `cli/testdata/`:
- `testdata/fixtures/simple-compose.yaml` — basic service, port, env vars
- `testdata/fixtures/with-database.yaml` — service with PostgreSQL dependency
- `testdata/fixtures/unsupported-features.yaml` — `build:`, custom networks
- `testdata/fixtures/unknown-service-type.yaml` — service with no catalog match
- `testdata/fixtures/empty-compose.yaml` — edge case: empty services block

**MUST** assert that the converter never modifies its `io.Reader` input (verify source bytes are unchanged after conversion).

**MUST** assert that sensitive variable names are redacted in exporter output. Test with variable names: `DB_PASSWORD`, `API_SECRET_KEY`, `OAUTH_TOKEN`, `AWS_ACCESS_KEY`.

**MUST NOT** use real Kubernetes cluster connections in unit tests. All Kubernetes API calls **MUST** be abstracted behind interfaces and mocked in tests.

### 5.2 CLI Integration Tests

**MUST** implement integration tests in `cli/integration_test.go` (build tag `//go:build integration`).

**MUST** test the full conversion pipeline end-to-end: read `docker-compose.yml` → produce `environment.yaml` → validate the output against the CRD schema.

**MUST** verify the 90 % mapping accuracy threshold (AC-052) by running the converter against a representative fixture containing 10 services and asserting ≥9 are correctly mapped.

**Definition — "correctly mapped":** A docker-compose service is **correctly mapped** when ALL of the following fields that are present in the source service definition are present and semantically equivalent in the generated IDP component:
- `image` → `spec.components[].type` resolves to the correct component type AND `imageTag` captures the tag
- `ports` → at least the container-side port is captured (host port mapping is documented as a warning, not a failure)
- `environment` → all non-secret key names are present; values for non-secret keys are preserved; keys whose names match the redaction pattern (`PASSWORD`, `SECRET`, `KEY`, `TOKEN`) are flagged, not silently dropped
- `volumes` → all mount paths are listed in the conversion report (IDP manages storage via Crossplane; the report documents volume paths for manual review)
- `depends_on` → dependency ordering is preserved in the component list order

A service mapped to `enabled: false` because no component type matches does **NOT** count toward the correctly-mapped total. A service is a mapping failure if any of the above fields are silently dropped without a report entry. (ref: BLK-002, NFR-011, AC-052)

**SHOULD** use `k3d` or `envtest` (`sigs.k8s.io/controller-runtime/pkg/envtest`) for integration tests that require a Kubernetes API server. **MUST NOT** run integration tests against a production or shared cluster.

### 5.3 CRD & Composition Validation

**MUST** validate all CRD YAML files with `kubectl apply --dry-run=client` as part of CI.

**MUST** validate all Crossplane Compositions with `crossplane beta validate` (or equivalent) in CI before merging to main.

**MUST** write at least one Composition unit test per component type using Crossplane's `crossplane beta render` command with an input XR fixture and asserting the expected rendered resources.

### 5.4 Acceptance Tests

**MUST** implement automated acceptance tests for each AC marked as High priority in [acceptance_criteria.md](acceptance_criteria.md) using a dedicated test cluster (not production).

**MUST** implement the following acceptance tests as a minimum viable set:

| AC | Test Description |
|---|---|
| AC-001 | Commit valid manifest → assert `Synced` within 5 min |
| AC-003 | Commit invalid manifest → assert zero resources created + human-readable ArgoCD sync failure |
| AC-005 | Disable component → assert all resources removed within 3 min |
| AC-007 | Push same manifest twice → assert no cluster changes on second push |
| AC-019 | Pod A → pod B cross-namespace connection → assert TCP refused |
| AC-037 | Delete Argo-managed Environment claim manually → assert ArgoCD restores within 3 min |
| AC-052 | Run converter on 10-service fixture → assert ≥9 correctly mapped |
| AC-059 | Export with PASSWORD/SECRET vars → assert values are `<REDACTED>` |
| AC-069 | Commit manifest with 5 components → assert `Synced` within 5 min |
| AC-070 | Disable component → assert resources gone within 3 min |

**MUST NOT** run acceptance tests against a shared or production cluster. Use a dedicated CI cluster or a local `k3d` cluster.

**Log retention tests** (AC-033, AC-035): **MUST** inject test events with a backdated `created_at` timestamp directly into the event store using a test helper function, bypassing the normal ArgoCD event path. The retention period **MUST** be configurable via environment variable `LOG_RETENTION_DAYS` (default: `7`); set to `0` (zero days, i.e., same-day expiry) in tests to validate expiry logic without waiting real days. (ref: MAJ-006)

**Concurrency test** (AC-072, NFR-004): **MUST** run on a cluster with minimum **3 nodes × 4 vCPU × 8 GB RAM**. Test procedure: create 50 environment manifests via 50 sequential Git commits within a 5-minute window using the CLI test helper. Assertion: median ArgoCD sync time ≤ 5 minutes; no individual environment exceeds 8 minutes (1.6× buffer). This test **MUST** run in a dedicated `load-test` CI job, not in the standard acceptance test suite. (ref: MAJ-007)

### 5.5 Backstage Plugin Tests

**MUST** write unit tests for any custom Backstage plugins or scaffolder actions using `@backstage/backend-test-utils`.

**MUST** write at least one integration test per scaffolder template that:
1. Renders the template with valid input values
2. Asserts the generated `environment.yaml` is a valid Environment claim matching the XRD-backed schema
3. Asserts that `owner` and `team` metadata are correctly populated

**SHOULD** test Backstage catalog ingestion by asserting that a catalog-info.yaml committed to Git is discoverable within 2 minutes (AC-020).

### 5.6 Security Tests

**MUST** include a test that asserts no committed YAML file in the `environments/` directory contains any of the following patterns: `password:`, `secret:`, `token:`, `privateKey:` as literal values (not references).

**MUST** run `kubescape scan` against all platform manifests in CI and fail the pipeline if any CRITICAL severity finding is introduced.

**SHOULD** run a NetworkPolicy conformance check using `netpol-verify` (or equivalent) to assert cross-namespace isolation holds for all environment namespace pairs. (ref: NFR-005, AC-041)

---

## 6. Cross-Cutting Constraints

### 6.1 Git Workflow

**MUST** protect the `main` branch: require at least one review approval and passing CI before merge.

**MUST** use conventional commits format: `feat:`, `fix:`, `chore:`, `docs:`, `test:`.

**MUST NOT** force-push to `main`. (ref: memory — Git Safety Protocol)

### 6.2 Observability

**MUST** emit Kubernetes Events for all state transitions in custom controllers or Compositions: `ProvisioningStarted`, `ProvisioningSucceeded`, `ProvisioningFailed`, `DeletionStarted`, `DeletionSucceeded`.

**MUST** expose a `/healthz` and `/readyz` endpoint for any custom service component introduced to the platform.

**SHOULD** add `prometheus.io/scrape: "true"` annotations to all platform-managed pods to enable metrics collection by the existing VictoriaMetrics stack. (ref: memory — VictoriaMetrics Stack)

### 6.3 Migration CLI Distribution

**MUST** build the CLI as a statically linked binary (`CGO_ENABLED=0`) for linux/amd64 and linux/arm64.

**MUST** publish the CLI binary as a GitHub Release asset with a checksum file (`sha256sums.txt`).

**MUST NOT** require Docker, Kubernetes access, or any runtime dependency to run the `idp migrate` and `idp export` commands. These **MUST** work on a developer's local machine with only the binary and a compose file.
