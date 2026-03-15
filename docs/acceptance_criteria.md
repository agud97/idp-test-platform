# Acceptance Criteria — IDP Test Environment Platform

**Version:** 1.1
**Date:** 2026-03-15
**Format:** WHEN – THEN – SHALL
**Sources:** [requirements.md](requirements.md) · [use_cases/](use_cases/)

---

## 1. Environment Lifecycle

### AC-001 · Environment creation — happy path
**WHEN** a developer commits a valid `kind: Environment` YAML manifest to the GitOps repository
**THEN** ArgoCD detects the commit and begins reconciliation
**SHALL** the environment reach `Synced` status within 5 minutes and all enabled components are running in a dedicated namespace

### AC-002 · Environment creation — maximum load
**WHEN** a developer commits an Environment manifest with 5 enabled components, each requiring a database
**THEN** ArgoCD reconciles all 10 resources in parallel
**SHALL** all components and databases reach healthy state within 5 minutes

### AC-003 · Environment creation — invalid YAML schema
**WHEN** a developer commits an Environment manifest that violates the claim/XRD schema
**THEN** the system attempts to apply the manifest
**SHALL** reject it with a human-readable validation error, expose the failure via the generated ArgoCD Application sync/operation state, and create zero Kubernetes resources for that environment

### AC-004 · Enable component — happy path
**WHEN** a developer changes a component flag from `enabled: false` to `enabled: true` and commits
**THEN** ArgoCD reconciles only the changed component
**SHALL** the Deployment, Service, and ConfigMap for that component exist and be ready within 5 minutes, while all other components remain unchanged

### AC-005 · Disable component — happy path
**WHEN** a developer changes a component flag from `enabled: true` to `enabled: false` and commits
**THEN** ArgoCD reconciles the change
**SHALL** all Kubernetes resources for that component (Deployment, Service, ConfigMap, DatabaseInstance, Secrets) be fully removed within 3 minutes, and all remaining components continue running without interruption

### AC-006 · Delete environment — full cleanup
**WHEN** a developer removes the environment manifest file from Git and pushes the commit
**THEN** ArgoCD detects the manifest removal
**SHALL** delete all Kubernetes resources, remove the namespace, and remove the catalog entry — all within 5 minutes of the push

### AC-007 · Idempotent apply — no-op on unchanged manifest
**WHEN** a developer pushes the identical environment manifest a second time without any changes
**THEN** ArgoCD performs a reconciliation cycle
**SHALL** make zero changes to the cluster and report status as `Synced`

### AC-008 · Manifest preserved after disable
**WHEN** a developer sets a component to `enabled: false` and pushes
**THEN** the system removes the component's resources
**SHALL** retain the component entry in the YAML manifest with `enabled: false`, so the component can be re-enabled without rewriting the manifest

---

## 2. Component Configuration Overrides

### AC-009 · Replica count override — valid value
**WHEN** a developer sets `replicas: 3` for a component and commits
**THEN** ArgoCD applies the change
**SHALL** the component Deployment have exactly 3 running and ready pods

### AC-010 · Image tag override
**WHEN** a developer sets `image_tag: v2.1.0` for a component and commits
**THEN** ArgoCD applies the rolling update
**SHALL** the component Deployment use the specified image tag, and the rollout complete without downtime to other components

### AC-011 · Replica count override — exceeds maximum
**WHEN** a developer sets `replicas: 51` (above the maximum of 50) and commits
**THEN** the system validates the manifest
**SHALL** reject it with an error message stating the allowed range (0–50), and apply no changes to the cluster

### AC-012 · Config override — unknown key
**WHEN** a developer includes an unrecognised key in `config_overrides` and commits
**THEN** the system validates the manifest
**SHALL** reject it with an error identifying the invalid key and the component type it was applied to, and apply no changes

### AC-013 · Replica set to zero — scale-down without resource removal
**WHEN** a developer sets `replicas: 0` for a component and commits
**THEN** ArgoCD applies the change
**SHALL** scale the Deployment to zero pods while keeping the Deployment, Service, ConfigMap, and DatabaseInstance resources intact

### AC-014 · Override scope — only target component affected
**WHEN** a developer overrides configuration for one component in a multi-component environment
**THEN** ArgoCD applies the change
**SHALL** update only the target component's resources; all other components remain at their current state

---

## 3. Database Provisioning

### AC-015 · Database provisioned alongside enabled component
**WHEN** a component with `supports_database: true` is set to `enabled: true` and reconciled
**THEN** the system provisions the database
**SHALL** a DatabaseInstance with status `Ready` exist in the namespace, and the component pod have database connection details injected via a Kubernetes Secret (not hardcoded in the manifest)

### AC-016 · Database removed with disabled component
**WHEN** a component that has an associated DatabaseInstance is set to `enabled: false` and committed
**THEN** the system removes the component resources
**SHALL** the DatabaseInstance and its credentials Secret be deleted within 3 minutes

### AC-017 · Unsupported database engine
**WHEN** a component specifies a database engine other than `postgresql` or `redis`
**THEN** the system validates the manifest
**SHALL** reject it with error "Unsupported database engine: `<engine>`" and create no Kubernetes resources

### AC-018 · Database credentials never exposed in manifest
**WHEN** a database is provisioned for a component
**THEN** the system creates database credentials
**SHALL** store them exclusively in a Kubernetes Secret; no credentials shall appear in ConfigMaps, Deployment environment variable literals, or Git-committed files

### AC-019 · Database inaccessible to other namespaces
**WHEN** a pod in environment A attempts a TCP connection to the database service in environment B
**THEN** the NetworkPolicy is evaluated
**SHALL** the connection be refused or time out; no data from environment B's database is accessible to environment A

---

## 4. Backstage Portal — Authentication

### AC-081 · OIDC sign-in page renders correctly
**WHEN** a user opens `http://backstage.idp.local:7007` in a browser
**THEN** the Backstage frontend loads
**SHALL** display a "Sign in with OIDC" button (not a "Enter as Guest" button); the guest sign-in option **shall not** be present

### AC-082 · OIDC login — happy path
**WHEN** a user clicks "Sign in with OIDC" and completes login on the Authentik page with valid credentials
**THEN** the OIDC popup closes and the parent window receives the identity
**SHALL** the user be signed in to Backstage, their display name visible in the top bar, and the catalog page accessible without further authentication prompts

### AC-083 · OIDC login — user entity present in catalog
**WHEN** a user completes OIDC authentication and Backstage attempts to resolve their identity
**THEN** the sign-in resolver queries the Backstage catalog for a `kind: User` entity matching the authenticated user's email
**SHALL** find the entity and complete sign-in successfully; if the entity is absent the error **shall** appear in Backstage backend logs and the user **shall** receive a clear sign-in failure message (not a silent redirect loop)

### AC-084 · Browser cache — new image served after rebuild
**WHEN** the Backstage Docker image is rebuilt and deployed
**THEN** a user opens Backstage in a browser that previously cached the old frontend JS
**SHALL** receive the new patched JS file (confirmed by a new filename, e.g., `module-backstage.oidcpatch.js`) without needing to manually clear the browser cache

---

## 4. Backstage Portal — Environment Management

### AC-020 · Create environment via portal — happy path
**WHEN** a developer selects a template, fills in all required fields, and submits the form in Backstage
**THEN** the system generates a manifest and commits it to Git
**SHALL** the environment appear in the Backstage catalog with correct owner and team metadata within 2 minutes, and reconciliation begin automatically

### AC-021 · Create environment via portal — missing required field
**WHEN** a developer submits the environment creation form with one or more required fields empty
**THEN** the system validates the form
**SHALL** display field-level error messages for each missing field and commit nothing to Git

### AC-022 · Create environment via portal — duplicate name
**WHEN** a developer submits a form with an environment name that already exists under their ownership
**THEN** the system validates the name
**SHALL** display "Environment name already exists. Choose a different name." and commit nothing to Git

### AC-023 · Create environment via portal — no templates available
**WHEN** a developer navigates to "New Environment" and no templates have been published
**THEN** the system loads the creation wizard
**SHALL** display "No templates available" and not present a broken or empty form

### AC-024 · Generated manifest matches template
**WHEN** a developer creates an environment from a template with specific parameter values
**THEN** the system generates the YAML
**SHALL** the committed manifest contain exactly the template structure with developer-supplied values substituted for all template parameters

### AC-085 · Kubernetes tab visible on environment entity page
**WHEN** a developer opens the catalog page of a deployed environment
**THEN** Backstage renders the entity page
**SHALL** display a "Kubernetes" tab showing live pods and deployments from the environment's namespace; the tab MUST be present without any manual configuration beyond the `backstage.io/kubernetes-cluster` and `backstage.io/kubernetes-namespace` annotations

---

## 5. Backstage Portal — Catalog & Status

### AC-025 · Browse catalog — access scope for developer
**WHEN** a developer views the Backstage catalog
**THEN** the system applies access rules
**SHALL** show only environments owned by that developer or belonging to their team; environments owned by other teams shall not appear

### AC-026 · Browse catalog — platform engineer access
**WHEN** a platform engineer views the Backstage catalog
**THEN** the system applies access rules
**SHALL** show all environments across all teams

### AC-027 · Browse catalog — filter by team
**WHEN** a user applies a team filter in the catalog
**THEN** the system filters the result set
**SHALL** show only environments belonging to the selected team, and hide all others

### AC-028 · View status — Synced environment
**WHEN** a developer opens the detail page of a healthy environment
**THEN** the system fetches status from ArgoCD
**SHALL** display `Synced` and a list of enabled components with their internal service endpoints

### AC-029 · View status — Degraded environment
**WHEN** a developer opens the detail page of a failed environment
**THEN** the system fetches status from ArgoCD
**SHALL** display a failed state summary using ArgoCD health when available or sync/operation error details otherwise, together with a human-readable error summary and a direct link to provisioning logs

### AC-030 · View status — data freshness
**WHEN** a developer views the environment status page
**THEN** the system fetches ArgoCD status
**SHALL** display status data no older than 30 seconds from the current ArgoCD sync state

### AC-031 · View status — ArgoCD unreachable
**WHEN** a developer opens an environment status page while ArgoCD is unavailable
**THEN** the system attempts to fetch status
**SHALL** display "Status unavailable — unable to reach reconciliation engine" rather than stale or incorrect data

---

## 6. Provisioning Logs

### AC-032 · View logs — owner access
**WHEN** the environment owner opens the provisioning logs tab
**THEN** the system fetches ArgoCD events
**SHALL** display a chronological list of events: resource creation, update, deletion, and error messages

### AC-033 · View logs — access denied for non-owner
**WHEN** a developer who does not own an environment attempts to view its provisioning logs
**THEN** the system checks ownership
**SHALL** display "Access denied — you are not the owner of this environment" and show no log data

### AC-034 · View logs — retention within 7 days
**WHEN** a developer views provisioning logs for an event that occurred 6 days ago
**THEN** the system fetches historical events
**SHALL** display the event with its full detail

### AC-035 · View logs — event older than 7 days
**WHEN** a developer views provisioning logs for an event that occurred 8 days ago
**THEN** the system fetches historical events
**SHALL** indicate that the event is outside the retention window and no longer available

### AC-036 · View logs — no events yet
**WHEN** a developer opens the logs tab for a newly submitted environment before ArgoCD has run
**THEN** the system queries for events
**SHALL** display "No provisioning events found yet" rather than an error

---

## 7. GitOps Reconciliation & Self-Healing

### AC-037 · Self-heal — manual resource deletion
**WHEN** a cluster administrator manually deletes an ArgoCD-managed `Environment` claim that is defined by the Git-managed environment manifest
**THEN** ArgoCD detects the drift in the next reconciliation cycle
**SHALL** recreate the claim within 3 minutes and restore the dependent runtime resources to the Git-defined state

### AC-038 · Self-heal — manual resource modification
**WHEN** someone manually modifies the ArgoCD-managed `Environment` claim via `kubectl` so that the desired replica count differs from Git
**THEN** ArgoCD detects the claim drift
**SHALL** restore the claim spec and resulting workload replica count to the values specified in the Git manifest within 3 minutes

### AC-039 · Audit trail — every change traceable in Git
**WHEN** any environment is created, modified, or deleted
**THEN** the change is reflected as a Git commit
**SHALL** the Git history contain the author identity, timestamp, and full diff of the manifest change

### AC-040 · Direct kubectl apply blocked
**WHEN** a developer applies a modified `Environment` claim directly via `kubectl apply` in the control namespace
**THEN** ArgoCD detects the out-of-sync state on the Argo-managed claim layer
**SHALL** revert the claim and dependent runtime resources to the Git-defined state within 3 minutes

---

## 8. Access Control & Namespace Isolation

### AC-041 · Namespace isolation — cross-environment network traffic
**WHEN** a pod in one environment namespace makes a TCP connection attempt to a pod in a different environment namespace
**THEN** the cluster NetworkPolicy is evaluated
**SHALL** the connection be refused or time out; no response from the target pod shall be received

### AC-042 · Namespace isolation — intra-environment traffic allowed
**WHEN** a pod makes a TCP connection to another pod within the same environment namespace
**THEN** the NetworkPolicy is evaluated
**SHALL** the connection succeed without restriction

### AC-043 · RBAC — unauthorised manifest modification
**WHEN** a developer attempts to push a commit modifying another developer's environment manifest
**THEN** the Git repository branch protection rules are evaluated
**SHALL** the push be rejected with an authorisation error and the manifest remain unchanged

### AC-044 · RBAC — unauthorised environment deletion
**WHEN** a developer attempts to delete an environment manifest that they do not own
**THEN** the repository access control is enforced
**SHALL** the deletion be rejected; no resources are removed from the cluster

---

## 9. Environment Templates

### AC-045 · Define template — happy path
**WHEN** a team lead submits a valid environment template with a unique name in Backstage
**THEN** the system validates and saves the template
**SHALL** the template appear in the Backstage scaffolder and be selectable by all team members immediately

### AC-046 · Define template — duplicate name
**WHEN** a team lead submits a template with a name that already exists
**THEN** the system validates the name
**SHALL** display "A template with this name already exists" and not save the new template

### AC-047 · Define template — references unknown component type
**WHEN** a team lead submits a template that references a component type not registered in the catalog
**THEN** the system validates the template body
**SHALL** reject it with "Unknown component type: `<name>`" and save nothing

### AC-048 · Template update — no impact on existing environments
**WHEN** a team lead updates an existing template
**THEN** the system saves the new version
**SHALL** all environments previously created from the template continue running unchanged; the update applies only to new environments created after the update

---

## 10. Component Type Registration

### AC-049 · Register component type — happy path
**WHEN** a platform engineer registers a new component type after its Crossplane Composition is deployed
**THEN** the system validates and saves the catalog entry
**SHALL** the component type be available for developers to reference in environment manifests immediately, with no platform downtime

### AC-050 · Register component type — no schema breaking change
**WHEN** a new component type is registered
**THEN** ArgoCD reconciles all existing environment manifests
**SHALL** all previously valid manifests continue to reconcile successfully without modification

### AC-051 · Register component type — duplicate name
**WHEN** a platform engineer attempts to register a component type with a name that already exists
**THEN** the system validates the catalog entry
**SHALL** reject it with "Component type `<name>` is already registered" and keep the existing entry unchanged

---

## 11. Migration — docker-compose Conversion

### AC-052 · Convert — happy path, 90 % mapping threshold
**WHEN** the CLI tool is run against a valid `docker-compose.yml` containing only supported service types
**THEN** the tool processes the file
**SHALL** output a valid IDP environment YAML with at least 90 % of service definitions correctly mapped, accompanied by a conversion report listing mapped, warned, and skipped items

### AC-053 · Convert — source file not modified
**WHEN** the CLI conversion tool is executed
**THEN** the tool reads the source `docker-compose.yml`
**SHALL** the original file remain byte-for-byte identical after the tool exits

### AC-054 · Convert — unsupported directive (`build:`)
**WHEN** the `docker-compose.yml` contains `build:` directives
**THEN** the tool processes the file
**SHALL** skip the `build:` directives and list them in a "Manual Review Required" section of the report without failing the conversion

### AC-055 · Convert — service with no matching component type
**WHEN** a docker-compose service name has no matching registered IDP component type
**THEN** the tool generates the output manifest
**SHALL** include the service as `enabled: false` with a `# TODO: No matching component type` comment and add a warning in the conversion report; the tool shall not exit with an error

### AC-056 · Convert — file not found
**WHEN** the CLI tool is invoked with a path to a non-existent file
**THEN** the tool attempts to read the file
**SHALL** exit immediately with a clear error message "File not found: `<path>`" and create no output file

---

## 12. Migration — Legacy Config Export

### AC-057 · Export — full access, complete output
**WHEN** the export tool is run against a legacy environment with full file and API access
**THEN** the tool extracts the configuration
**SHALL** produce an output file containing service names, container images, exposed ports, environment variables, volume mount paths, and inter-service dependencies for all services

### AC-058 · Export — restricted file access, containers running
**WHEN** the `docker-compose.yml` is inaccessible but the environment's containers are running
**THEN** the tool switches to container inspection mode
**SHALL** extract configuration from running container metadata and mark each inferred field as "inferred from runtime" in the export summary

### AC-059 · Export — sensitive environment variable redaction
**WHEN** the export tool encounters environment variable names containing `PASSWORD`, `SECRET`, `KEY`, or `TOKEN`
**THEN** the tool writes the output file
**SHALL** replace the values with `<REDACTED>` and list each redacted variable in a "Secrets — Manual Re-entry Required" section

### AC-060 · Export — non-destructive operation
**WHEN** the export tool is run against any legacy environment
**THEN** the tool accesses the environment
**SHALL** not stop, restart, modify, or send write-type requests to any service in the legacy environment

### AC-061 · Export — partial access, no silent failures
**WHEN** only a subset of legacy services are accessible during export
**THEN** the tool completes the export
**SHALL** export all accessible services and explicitly list all inaccessible services in an "Incomplete Export" section; the tool shall not silently skip missing data

### AC-062 · Export — total access failure
**WHEN** the export tool has no access to the legacy environment (no file, no API, no containers)
**THEN** the tool attempts to connect
**SHALL** exit with error "Cannot access legacy environment `<name>`. No data extracted." and create no output file

---

## 13. Migration — Validation & Tracking

### AC-063 · Parallel validation — legacy environment unaffected
**WHEN** a developer runs a full validation test suite against the migrated IDP environment
**THEN** the test suite sends traffic to both environments
**SHALL** the legacy environment remain fully operational and all its services continue responding normally throughout the validation run

### AC-064 · Validation — status update on success
**WHEN** a developer marks an environment as validated in the migration tracker
**THEN** the system updates the record
**SHALL** set `LegacyEnvironment.migration_status` to `validated`; the team's derived migrated count (computed from `LEGACY_ENVIRONMENT` records) increases accordingly with no separate counter to update

### AC-065 · Migration tracking dashboard — per-team summary
**WHEN** a platform engineer opens the migration tracking dashboard
**THEN** the system aggregates LegacyEnvironment records
**SHALL** display a table with one row per team showing: total legacy environments, migrated, validated, blocked, and overall migration percentage

### AC-066 · Deprecation gate — blocked while incomplete
**WHEN** a platform engineer initiates the legacy platform deprecation workflow
**THEN** the system evaluates all LegacyEnvironment records
**SHALL** block the deprecation and display a list of environments whose `migration_status` is not `completed`

### AC-067 · Deprecation gate — allowed only at 100 %
**WHEN** every LegacyEnvironment record has `migration_status: completed`
**THEN** the platform engineer initiates the deprecation workflow
**SHALL** the system permit the deprecation to proceed

### AC-073 · Register legacy environment — happy path
**WHEN** a platform engineer runs `idp register-legacy --name <name> --team <team>`
**THEN** the CLI creates a LEGACY_ENVIRONMENT record
**SHALL** the record exist with `migration_status: pending`, the correct `team_id`, and the environment appear in the migration dashboard total count for that team

### AC-074 · Register legacy environment — team not found
**WHEN** a platform engineer runs `idp register-legacy` with a team name that has no TEAM record
**THEN** the CLI validates the team
**SHALL** exit with error "Team `<name>` not found" and create no record

### AC-075 · Register legacy environment — duplicate registration
**WHEN** a platform engineer attempts to register an environment that is already registered for the same team
**THEN** the CLI checks for an existing record
**SHALL** exit with error "Legacy environment `<name>` is already registered for team `<team>`" and create no duplicate record

### AC-076 · Mark migration complete — happy path
**WHEN** a platform engineer runs `idp complete-migration --env-id <id>` for an environment with `migration_status: validated`
**THEN** the CLI updates the record
**SHALL** set `migration_status: completed` with a `completed_at` timestamp, and the environment counts toward the deprecation gate threshold

### AC-077 · Mark migration complete — not validated
**WHEN** a platform engineer runs `idp complete-migration` for an environment whose `migration_status` is not `validated`
**THEN** the CLI validates the precondition
**SHALL** exit with error stating the current status and instruct to complete validation first; no status change is applied

### AC-078 · Initiate deprecation — gate passes at 100%
**WHEN** a platform engineer runs `idp deprecate-legacy --confirm` and all registered LEGACY_ENVIRONMENT records have `migration_status: completed`
**THEN** the CLI generates the deprecation report
**SHALL** write `deprecation-report-<date>.md` listing all environments with completion timestamps, set the `LEGACY_PLATFORM_DEPRECATED=true` flag, and exit with success

### AC-079 · Initiate deprecation — gate blocked
**WHEN** a platform engineer runs `idp deprecate-legacy --dry-run` and at least one LEGACY_ENVIRONMENT has `migration_status != completed`
**THEN** the CLI evaluates all records
**SHALL** display "Gate FAILED" with a list of blocking environments (name, team, status), generate no report, and set no deprecated flag

### AC-080 · Initiate deprecation — no environments registered
**WHEN** a platform engineer runs `idp deprecate-legacy` with zero LEGACY_ENVIRONMENT records in the tracker
**THEN** the CLI checks record count
**SHALL** exit with error "No legacy environments registered" and perform no further action

### AC-068 · Runbook — missing application type
**WHEN** a developer opens the migration documentation portal to find a runbook for their application type
**THEN** no runbook exists for that type
**SHALL** the portal display a "Runbook missing — contact platform engineer" notice rather than a 404 or blank page

---

## 14. Performance SLAs

### AC-069 · Environment creation SLA — 5 minutes
**WHEN** a valid Environment manifest with up to 5 enabled components is committed to Git
**THEN** ArgoCD begins reconciliation
**SHALL** the environment reach `Synced` status within 5 minutes of the push timestamp

### AC-070 · Component teardown SLA — 3 minutes
**WHEN** a component is disabled in the manifest and committed
**THEN** ArgoCD detects the change
**SHALL** all Kubernetes resources for the disabled component be fully removed within 3 minutes of the reconciliation cycle detecting the change

### AC-071 · Portal response time — normal load
**WHEN** up to 20 concurrent users access any page in the Backstage portal
**THEN** the portal processes the requests
**SHALL** every page load complete within 3 seconds

### AC-072 · Concurrent environments — no SLA degradation
**WHEN** 50 environments are simultaneously active and being reconciled in the cluster
**THEN** ArgoCD processes all environments
**SHALL** no individual environment exceed the 5-minute creation SLA or the 3-minute teardown SLA due to resource contention
