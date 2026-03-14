# Specification Review — IDP Test Environment Platform

**Version:** 1.1
**Date:** 2026-03-14
**Resolution applied:** 2026-03-14 — all 18 issues resolved (see resolution notes in each section)
**Reviewer role:** Specification Reviewer / QA Architect
**Reviewed files:** `vision.md`, `requirements.md`, `entity_model.md`, `use_cases/*.md`, `use_cases.md`, `acceptance_criteria.md`, `constraints.md`

**Issue count:** 4 BLOCKER · 7 MAJOR · 7 MINOR

---

## BLOCKERS — Must resolve before implementation starts

---

### BLK-001 · Direct contradiction: `enabled: false` behavior
> ✅ **RESOLVED** — `constraints.md §2.2` updated: resources ARE deleted on `enabled:false`. `replicas:0` documented as separate concept (scale-down, no deletion).

**Files:** `constraints.md §2.2` vs `UC-003` vs `AC-005`

**Contradiction:**

| Source | States |
|---|---|
| `constraints.md §2.2` | "MUST NOT delete the Deployment or Service resource when `enabled: false`. Resource removal is handled by ArgoCD pruning when the component entry is **removed from the manifest entirely**." |
| `UC-003 Main Success Scenario step 5` | "Crossplane marks all resources of the component for **deletion** (Deployment, Service, ConfigMap)" |
| `AC-005` | "all Kubernetes resources for that component… **fully removed** within 3 minutes" |

The constraint says resources survive `enabled: false` (only scaled to 0). The use case and acceptance criteria say resources are deleted. An agent implementing `constraints.md` will fail AC-005 and UC-003. An agent implementing UC-003 will violate constraint 2.2.

**Resolution required:** Choose one of:
- **Option A (preferred):** Resources ARE deleted on `enabled: false`. Update `constraints.md §2.2` to remove the MUST NOT. Crossplane must delete managed resources when the XR field changes.
- **Option B:** Resources are only scaled to 0 on `enabled: false`. Update UC-003 and AC-005 to say "scaled to 0" instead of "removed". Add a new concept (`suspended`) to the lifecycle. Update COMPONENT `status` field in entity model accordingly.

---

### BLK-002 · "Correctly mapped" is unmeasurable
> ✅ **RESOLVED** — Formal definition added to `constraints.md §5.2`: 5 fields checked (image, ports, environment vars, volumes, depends_on), with specific rules for each. Added to `requirements.md` definition block.

**Files:** `NFR-011`, `AC-052`, `constraints.md §5.2`

All three documents reference a 90% mapping accuracy threshold but none define what "correctly mapped" means for a single service.

**Examples of ambiguity:**
- Is a service "correctly mapped" if its `image` field is transferred but `volumes` are skipped?
- Does mapping `ports: ["8080:80"]` to `service.port: 80` count as correct even though the host port is lost?
- Does a service mapped to `enabled: false` (no component type found) count toward the 90% or not?

The CLI tool cannot be validated against AC-052 without this definition, and the test in `constraints.md §5.2` ("assert ≥9/10 services correctly mapped") cannot be written.

**Resolution required:** Add a definition block to `acceptance_criteria.md` and `constraints.md §5.2`:
> A service is **correctly mapped** when all of the following fields that are present in the source are present and semantically equivalent in the output: `image`, `ports` (container port), `environment` variables (keys and non-secret values), `volumes` (mount paths), `depends_on` (dependency list). Missing optional fields produce a warning, not a mapping failure.

---

### BLK-003 · `validated → completed` transition is undefined
> ✅ **RESOLVED** — Added FR-021, UC-021 (`idp complete-migration` CLI command). `completed` state is set by platform engineer after legacy env is shut down. Added AC-076/077. `TEAM.migrated_envs` counter removed (MIN-004 co-resolved).

**Files:** `entity_model.md (LEGACY_ENVIRONMENT.migration_status)`, `UC-017`, `UC-019`, `AC-064`, `AC-066`, `AC-067`

The entity model defines five `migration_status` values: `pending → in_progress → validated → completed → blocked`. AC-064 sets status to `validated` after parallel validation passes. AC-066/067 require `completed` for the deprecation gate. But:

- No UC describes who triggers the `validated → completed` transition
- No FR defines an action "mark migration as complete"
- No AC tests the `validated → completed` transition
- `TEAM.migrated_envs` is incremented "on validated" per AC-064, but `migrated_envs` semantically implies completion, not just validation

**Consequence:** The deprecation gate (AC-066, AC-067) can never be reached because no process sets `migration_status: completed`.

**Resolution required:** Either:
- Add FR-020 "Mark migration as completed" + UC-020 with actor Platform Engineer: triggered after the legacy environment is decommissioned (cutover done). This is the post-validation step.
- Or collapse `validated` and `completed` into one state if there is no distinct cutover step.

Also clarify whether `TEAM.migrated_envs` increments on `validated` or `completed`.

---

### BLK-004 · Deprecation workflow has no implementation spec
> ✅ **RESOLVED** — Added FR-022, UC-022 (`idp deprecate-legacy --dry-run / --confirm` CLI command with report generation). Added AC-078/079/080.

**Files:** `AC-066`, `AC-067`, `C-012`, `UC-019`

AC-066 says "a platform engineer initiates the legacy platform deprecation workflow" and AC-067 says "the system permits the deprecation to proceed." But:

- No FR, UC, or entity model field describes what the deprecation workflow IS
- Is it a button in Backstage? A CLI command? A Git commit? A CRD field?
- No spec defines the observable outcome of a successful deprecation (what does "decommission" mean technically?)
- No error path is defined if deprecation is attempted while environments are still `pending`

**Consequence:** Acceptance criteria AC-066 and AC-067 are untestable because there is no triggerable action to test against.

**Resolution required:** Add FR-020 (or FR-021 if BLK-003 is resolved separately): "Initiate legacy platform deprecation" with actor Platform Engineer. Define the trigger (e.g., a CLI command `idp deprecate-legacy`), the gate check (all `migration_status: completed`), and the outcome (e.g., a LEGACY_ENVIRONMENT `archived_at` timestamp or a platform-level lock flag).

---

## MAJOR — Must resolve before the affected component is implemented

---

### MAJ-001 · Wrong ArgoCD ApplicationSet generator type
> ✅ **RESOLVED** — `constraints.md §2.3` updated to use `git files` generator with glob `environments/**/*.yaml`, with corrected YAML example.

**File:** `constraints.md §2.3`

**Text:** "MUST use an `ApplicationSet` with a `git` generator (**directory strategy**)"

**Problem:** The ArgoCD ApplicationSet `git` generator has two sub-strategies:
- `directories` — creates one Application per **directory** in the repo
- `files` — creates one Application per **file** matching a glob pattern

Environments are stored as files: `environments/<team>/<env-name>.yaml`. Using the `directories` generator would create one Application per team directory, not per environment file. This would break the one-Application-per-environment model required by FR-013 and UC-001.

**Resolution:** Replace in `constraints.md §2.3`:
```yaml
# WRONG
generators:
  - git:
      directories:
        - path: environments/*/*

# CORRECT
generators:
  - git:
      files:
        - path: environments/**/*.yaml
```

---

### MAJ-002 · No UC or FR for registering legacy environments
> ✅ **RESOLVED** — Added FR-020, UC-020 (`idp register-legacy` CLI). Added AC-073/074/075.

**Files:** `UC-019 (Preconditions)`, `AC-065`, `AC-066`, entity_model `LEGACY_ENVIRONMENT`

UC-019 precondition states: "Legacy environments have been registered in the migration tracker." AC-065 requires the dashboard to show per-team data. But:

- No FR describes who registers legacy environments or how
- No UC for "Register Legacy Environment" exists
- No CLI command, API endpoint, or UI form is specified for creating `LEGACY_ENVIRONMENT` records
- `TEAM.total_legacy_envs` cannot be populated without a registration step

**Consequence:** The migration tracking dashboard (AC-065) will show empty data, and the deprecation gate (AC-066) can never evaluate accurately.

**Resolution:** Add FR-020 (or renumber after BLK-003/004): "Register legacy environment for migration tracking" with actor Platform Engineer. Define the input (environment name, team, optional compose file path) and the outcome (LEGACY_ENVIRONMENT record created with `migration_status: pending`). Add a corresponding CLI command `idp register-legacy`.

---

### MAJ-003 · Migration tracking dashboard is unspecified
> ✅ **RESOLVED** — `constraints.md §3.3` specifies: custom Backstage plugin `@internal/plugin-migration-dashboard`, route `/migration`, backend API `GET /api/migration/status`, 60-second refresh.

**Files:** `UC-019`, `AC-065`, `FR-019`

FR-019 says "track which teams have migrated" and UC-019 references "migration tracking dashboard." AC-065 describes what it shows. But no spec defines:

- What component implements the dashboard (Backstage plugin? Grafana panel? Custom React page?)
- What URL or navigation path leads to it
- Whether it requires a new Backstage plugin, an existing plugin extension, or a standalone service
- Whether it is real-time or batch-refreshed

**Consequence:** An implementing agent cannot build the dashboard without arbitrarily picking a technology and structure.

**Resolution:** Add a subsection to `constraints.md §3.3` specifying the implementation approach. Recommended: extend the Backstage catalog with a custom `MigrationStatusPage` plugin that queries a backend aggregation API. Define the API contract (endpoint, response shape) or confirm the data comes directly from querying the `LEGACY_ENVIRONMENT` entity store.

---

### MAJ-004 · "Application type" for runbook coverage is undefined
> ✅ **RESOLVED** — Definition added to `requirements.md` before NFR table: application type = unique combination of component types in at least one legacy environment, enumerated during FR-020 registration.

**Files:** `NFR-012`, `AC-068`, `UC-018 BR-001`

NFR-012 requires "100% of application types currently running on the legacy platform" to be covered by runbooks. AC-068 tests for a missing runbook by "application type." But "application type" is never defined.

**Possible interpretations:**
1. = `ComponentType` (one runbook per component type: `postgresql`, `redis`, `webapp`)
2. = a common stack combination (e.g., "webapp + postgresql", "worker + redis")
3. = each distinct docker-compose topology observed in the legacy platform

These interpretations lead to very different numbers of runbooks (3 vs 10+ vs 50+).

**Resolution:** Add a definition to `requirements.md` or `constraints.md`: "An **application type** is defined as a unique combination of component types used together in at least one legacy environment (e.g., `[webapp, postgresql]`). The platform engineer must enumerate application types from the legacy platform inventory during the registration step (see MAJ-002)."

---

### MAJ-005 · USER.team_id NOT NULL breaks platform engineer model
> ✅ **RESOLVED** — `entity_model.md USER`: `team_id` changed to Optional. Added constraint: Not Null for developer/qa_engineer/team_lead, Null for platform_engineer.

**File:** `entity_model.md (USER)`

`USER.team_id` is defined as `Not Null, Foreign Key (TEAM.id)`. But platform engineers govern all teams and should not be scoped to a single team. The access rules in `AC-026` state platform engineers see ALL environments — implying they are not team-scoped.

**Problem:** Creating a `platform_engineer` USER record requires assigning them to a specific team, which contradicts the access model. If they're assigned to team "platform", that team would incorrectly appear in `TEAM.total_legacy_envs` counts and environment ownership queries.

**Resolution:** Make `USER.team_id` Optional (nullable) for users with `role: platform_engineer`. Add a constraint: "team_id must be Not Null when role is developer, qa_engineer, or team_lead. team_id is Null when role is platform_engineer."

---

### MAJ-006 · Log retention test has no strategy
> ✅ **RESOLVED** — `constraints.md §5.4`: inject backdated events via test helper; `LOG_RETENTION_DAYS` env var configurable; set to `0` in tests to validate expiry without waiting real days.

**Files:** `AC-033` (6-day retention must succeed), `AC-035` (8-day retention must fail)

These ACs require testing time-based data expiry, but no test strategy is defined:
- Waiting 6–8 real days in CI is impractical
- The test data setup is not described
- No test fixture format or backdating mechanism is specified in `constraints.md §5`

**Consequence:** AC-033 and AC-035 are in the mandatory acceptance test set but cannot be automated as written.

**Resolution:** Add to `constraints.md §5.4`: "For log retention tests, inject a test event with a backdated `created_at` timestamp directly into the event store (bypassing the normal ArgoCD event path). Use a configurable retention period environment variable (`LOG_RETENTION_DAYS`) that can be set to `0d2h` in the test environment to allow time-based tests to complete in minutes."

---

### MAJ-007 · Concurrency test (50 environments) has no infrastructure spec
> ✅ **RESOLVED** — `constraints.md §5.4`: minimum 3 nodes × 4 vCPU × 8 GB RAM, 50 sequential commits in 5 min window, assertion: median ≤5 min / max ≤8 min, runs in dedicated `load-test` CI job.

**Files:** `NFR-004`, `AC-072`, `constraints.md §5.4`

AC-072 is listed as a mandatory acceptance test but no spec defines:
- Minimum cluster size (node count, CPU, memory) required to run 50 environments without SLA degradation
- How the test creates 50 environments (sequential commits? batch script? parallel API calls?)
- What "degraded reconciliation latency" means in measurable terms for the test assertion
- Whether this test runs in CI or only in a dedicated load-test environment

**Consequence:** An agent implementing AC-072 cannot determine whether its test environment is valid or whether a latency breach is a platform limitation or a test setup issue.

**Resolution:** Add to `constraints.md §5.4`: "The 50-environment concurrency test MUST run on a cluster with minimum 3 nodes × 4 CPU × 8 GB RAM. Environments MUST be created via 50 sequential Git commits within a 5-minute window. The assertion is: median ArgoCD sync time across all 50 environments ≤ 5 minutes, with no individual environment exceeding 8 minutes (60% buffer over SLA)."

---

## MINOR — Should resolve before affected feature is complete

---

### MIN-001 · Namespace length overflow has no truncation strategy
> ✅ **RESOLVED** — `constraints.md §4.1`: truncation formula added with 4-char hex hash suffix for uniqueness. Go implementation sketch included.

**Files:** `constraints.md §4.1`, `entity_model.md (ENVIRONMENT.namespace)`

Namespace pattern is `env-<team>-<env-name>`. The entity model limits namespace to 63 characters (Kubernetes constraint). With a team name of 20 chars and environment name of 30 chars, the derived namespace would be `env-` (4) + 20 + `-` (1) + 30 = 55 chars — within limit. But edge cases exist (team: 25, name: 35 → 65 chars, which exceeds the limit).

**Resolution:** Add to `constraints.md §4.1`: "If the derived namespace name exceeds 63 characters, truncate `env-name` to fit and append a 4-character hash of the full name to preserve uniqueness: `env-<team[:20]>-<name[:N]>-<hash4>`."

---

### MIN-002 · ENVIRONMENT_TEMPLATE has no `parameters` field
> ✅ **RESOLVED** — `entity_model.md ENVIRONMENT_TEMPLATE`: added `parameters: String 4000, Optional` with JSON schema `{name, type, description, required, defaultValue}`. Uniqueness constraint added.

**Files:** `entity_model.md (ENVIRONMENT_TEMPLATE)`, `UC-009`, `constraints.md §4.2`

UC-009 step 5 says "Developer defines template parameters (e.g., `environment_name`, `image_tag`)." Constraints say "any value developers are expected to customise MUST be exposed as a named template parameter." But the entity model stores only `template_yaml: String 65535` with no separate `parameters` structure.

**Consequence:** There is no way to enumerate valid parameters for a template, validate input values against parameter types/constraints, or display a typed form in the Backstage UI.

**Resolution:** Add a `parameters` field to `ENVIRONMENT_TEMPLATE`: `String 4000, Optional` (stored as JSON array of `{name, type, description, required, defaultValue}` objects). Update the entity model.

---

### MIN-003 · Backstage authentication method is unspecified
> ✅ **RESOLVED** — `constraints.md §3.3`: Backstage uses Authentik OIDC provider, user role/team from LDAP group claims, guest auth disabled in production.

**Files:** `NFR-006`, `AC-025`, `AC-026`, `UC-006`

NFR-006 requires RBAC enforcement and AC-025/026 define different catalog views per role. But no spec defines:
- How users authenticate to Backstage (guest? OAuth? LDAP?)
- How Backstage determines a user's `role` and `team_id`
- Whether Backstage integrates with the existing Authentik/OpenLDAP SSO stack on the cluster

Without authentication, access scoping (AC-025, AC-026) cannot be implemented.

**Resolution:** Add to `constraints.md §3.3`: "Backstage MUST use Authentik as its OAuth2 provider via the existing `openwebui` OAuth2 provider pattern (or a new Authentik application). User `role` and `team` claims MUST be sourced from LDAP group membership synced into Authentik. The Backstage `identity` plugin MUST map LDAP groups to platform roles."

---

### MIN-004 · TEAM.migrated_envs counter has two sources of truth
> ✅ **RESOLVED** — `entity_model.md TEAM`: removed `total_legacy_envs` and `migrated_envs` columns entirely. Both values are now derived by aggregating `LEGACY_ENVIRONMENT` records at query time.

**Files:** `entity_model.md (TEAM)`, `AC-064`

`TEAM.migrated_envs` is a denormalised counter. AC-064 says it is incremented when `migration_status` transitions to `validated`. But the count can also be derived by querying `COUNT(LEGACY_ENVIRONMENT WHERE team_id=X AND migration_status IN ('validated','completed'))`.

Two sources of truth will diverge if records are updated directly (e.g., bulk migration or data corrections) without going through the AC-064 path.

**Resolution:** Either remove `TEAM.migrated_envs` from the entity model (derive it from `LEGACY_ENVIRONMENT` at query time), or add a constraint: "TEAM.migrated_envs MUST always equal the count of LEGACY_ENVIRONMENT records for that team with migration_status IN ('validated', 'completed'). The system MUST enforce this via a transactional update whenever LEGACY_ENVIRONMENT.migration_status changes."

---

### MIN-005 · docker-compose format version minimum is undefined
> ✅ **RESOLVED** — `constraints.md §3.2`: Compose Specification v1.0+ (version 3.x and unversioned) MUST be supported; version 2.x SHOULD be supported via backward-compat loader; unsupported directives go to report.

**File:** `constraints.md §3.2`

The constraint specifies `github.com/compose-spec/compose-go` as the parser library but does not state which docker-compose format versions are supported as input. The legacy platform uses docker-compose files that may be `version: "2.x"`, `"3.x"`, or unversioned (Compose Specification).

**Resolution:** Add to `constraints.md §3.2`: "The converter MUST support docker-compose files conforming to the Docker Compose Specification (compose-spec v1.0+), which covers `version: "3.x"` and unversioned files. Files using `version: "2.x"` syntax SHOULD be supported; unsupported `version: "2.x"` directives MUST be listed in the conversion report."

---

### MIN-006 · Template ownership succession is undefined
> ✅ **RESOLVED** — `entity_model.md ENVIRONMENT_TEMPLATE`: added constraint requiring ownership transfer to another team_lead or platform_engineer before owner deactivation.

**Files:** `entity_model.md (ENVIRONMENT_TEMPLATE)`, `UC-009 BR-002`

`ENVIRONMENT_TEMPLATE.owner_id` is `Not Null`. If the owning team lead leaves the organisation and their USER record is deactivated or deleted, all templates they own become ownerless (FK violation) or inaccessible.

**Resolution:** Add to `entity_model.md (ENVIRONMENT_TEMPLATE)`: "If the owner user is deactivated, ownership MUST be transferred to another user with `role: team_lead` from the same team, or to a platform engineer. A deactivated user MUST NOT be the sole owner of an active template."

---

### MIN-007 · DatabaseInstance has no namespace reference
> ✅ **RESOLVED** — `entity_model.md DATABASE_INSTANCE`: added `namespace: String 63, Not Null`, denormalised from ENVIRONMENT, with constraint enforcing equality.

**File:** `entity_model.md (DATABASE_INSTANCE)`

A `DatabaseInstance` is a Kubernetes resource and lives in a specific namespace. However, the entity model provides no direct `namespace` field. To find the namespace of a DatabaseInstance, an agent must traverse: `DATABASE_INSTANCE → COMPONENT.environment_id → ENVIRONMENT.namespace` — a 3-entity join.

This makes it difficult to implement operations like "list all databases in namespace X" without joins, and increases the risk of accidentally operating on the wrong namespace.

**Resolution:** Add `namespace: String 63, Not Null` to `DATABASE_INSTANCE`, populated at creation time from `ENVIRONMENT.namespace`. Add a constraint: "namespace must equal the namespace of the Environment that owns the parent Component."

---

## Checklist Summary

| Category | Status | Notes |
|---|---|---|
| Every AC has a clear test strategy | ✅ PASS | BLK-002, MAJ-006, MAJ-007 resolved |
| All error scenarios have defined behavior | ✅ PASS | BLK-004 resolved with UC-022 alt flows |
| Edge cases explicitly addressed | ✅ PASS | MIN-001, MIN-005 resolved |
| Performance requirements are measurable | ✅ PASS | MAJ-007 resolved with infra spec |
| No contradictions between AC and constraints | ✅ PASS | BLK-001 resolved |
| Package structure supports all components | ✅ PASS | MAJ-003 resolved |
| Data types consistent throughout | ✅ PASS | MIN-002, MIN-007 resolved |
| Technical constraints are specific enough | ✅ PASS | MAJ-001, MAJ-003 resolved |
| No circular dependencies | ✅ PASS | — |
| All external dependencies identified | ✅ PASS | MIN-003, MIN-005 resolved |
| Each AC maps to at least one test case | ✅ PASS | BLK-003/004 resolved with AC-073–080 |
| Test data requirements are clear | ✅ PASS | MAJ-006, MAJ-007 resolved |
| Success/failure conditions are unambiguous | ✅ PASS | BLK-002 resolved |

---

## Recommended Resolution Order

1. **BLK-001** — resolve first; affects Crossplane implementation, UC-003, and 6 acceptance criteria
2. **BLK-002** — resolve before CLI implementation starts; blocks all converter tests
3. **BLK-003 + BLK-004** — resolve together; both concern the migration lifecycle end state
4. **MAJ-001** — fix ArgoCD config before any infrastructure is provisioned
5. **MAJ-002** — required before migration tracking feature can be built
6. **MAJ-005** — fix entity model before any database schema is generated
7. **MAJ-003 + MAJ-004** — resolve before migration sprint begins
8. **MAJ-006 + MAJ-007** — resolve before acceptance test suite is written
9. **MIN-001 through MIN-007** — resolve during implementation of the relevant feature
