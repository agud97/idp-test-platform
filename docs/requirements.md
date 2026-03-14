# Requirements Catalog — IDP Test Environment Platform

**Version:** 1.0
**Date:** 2026-03-14
**Source:** [docs/vision.md](vision.md)

---

## Functional Requirements

| ID     | Title                           | User Story                                                                                                                                              | Priority | Status |
|--------|---------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------|----------|--------|
| FR-001 | Submit environment manifest     | As a developer, I want to submit a single `kind: Environment` YAML claim to create a test environment so that I can provision everything I need without writing Kubernetes manifests. | High     | Open   |
| FR-002 | Enable application component    | As a developer, I want to set `enabled: true` for an application in my YAML so that all its Kubernetes resources (namespace, deployments, services, databases) are created automatically. | High     | Open   |
| FR-003 | Disable application component   | As a developer, I want to set `enabled: false` for an application in my YAML so that all its Kubernetes resources are removed automatically.             | High     | Open   |
| FR-004 | Provision database per app      | As a developer, I want a database (PostgreSQL or Redis) to be provisioned alongside my application when enabled so that I do not need to manage database infrastructure manually. | High     | Open   |
| FR-005 | Create environment via portal   | As a developer, I want to create a new environment through the Backstage UI by filling a template form so that I do not need to write YAML from scratch.  | High     | Open   |
| FR-006 | Browse environment catalog      | As a developer, I want to see all my active environments in the Backstage software catalog so that I can track what is running and who owns it.           | Medium   | Open   |
| FR-007 | View environment status         | As a developer, I want to see the provisioning status (Synced / Progressing / Degraded) of my environment in the portal so that I know if my environment is ready to use. | High     | Open   |
| FR-008 | Delete environment              | As a developer, I want to delete an environment by removing or archiving its manifest in Git so that all associated resources are cleaned up automatically. | High     | Open   |
| FR-009 | Define environment templates    | As a team lead, I want to define reusable environment templates in Backstage so that developers on my team can provision consistent, pre-approved stacks.  | Medium   | Open   |
| FR-010 | Register new component type     | As a platform engineer, I want to add a new application component type to the platform catalog so that developers can include it in their environment YAML without platform changes to the schema. | Medium   | Open   |
| FR-011 | Isolate environments per namespace | As a QA engineer, I want each test environment to run in a dedicated Kubernetes namespace so that environments do not interfere with each other.        | High     | Open   |
| FR-012 | Override component configuration | As a developer, I want to override default configuration values (e.g., replica count, image tag) for a component in my YAML so that I can test non-default scenarios. | Medium   | Open   |
| FR-013 | GitOps reconciliation           | As a platform engineer, I want committed Environment claims to be continuously reconciled from Git by ArgoCD and Crossplane so that any manual drift is corrected automatically.    | High     | Open   |
| FR-014 | View provisioning logs          | As a developer, I want to view provisioning events and error messages in the portal so that I can diagnose failures without accessing Kubernetes directly. | Medium   | Open   |
| FR-015 | Convert docker-compose to IDP manifest | As a developer, I want a CLI tool that converts an existing `docker-compose.yml` into an IDP environment YAML so that I can migrate without rewriting the configuration from scratch. | High     | Open   |
| FR-016 | Export legacy environment config | As a platform engineer, I want to export the configuration of a legacy docker-compose environment so that its services, environment variables, and volumes are available for migration even when direct access is restricted. | High     | Open   |
| FR-017 | Validate migrated environment   | As a developer, I want to run the migrated IDP environment in parallel with the legacy docker-compose environment so that I can verify functional equivalence before switching over. | High     | Open   |
| FR-018 | Follow migration runbook        | As a developer, I want a step-by-step migration guide for each application type so that I can migrate my environment independently without platform engineer involvement. | Medium   | Open   |
| FR-019 | Track migration status per team | As a platform engineer, I want to track which teams have migrated to the new platform so that I can plan the deprecation of the legacy platform. | Medium   | Open   |
| FR-020 | Register legacy environment     | As a platform engineer, I want to register a legacy docker-compose environment in the migration tracker so that its migration progress can be monitored and the team's total count is accurate. | High     | Open   |
| FR-021 | Mark migration as complete      | As a platform engineer, I want to mark a validated legacy environment as `completed` after its legacy instance has been decommissioned so that the deprecation gate can be evaluated. | High     | Open   |
| FR-022 | Initiate legacy platform deprecation | As a platform engineer, I want to trigger a deprecation check that gates legacy platform shutdown on 100% of environments being `completed` so that no active environment is lost during decommissioning. | High     | Open   |

---

> **Definition — Application Type** (used in FR-018, NFR-012): An **application type** is a unique combination of component types used together in at least one legacy environment (e.g., `[webapp, postgresql]`, `[worker, redis]`, `[webapp, postgresql, redis]`). The platform engineer enumerates application types from the legacy platform inventory during the FR-020 registration step. Each distinct combination requires its own migration runbook.

## Non-Functional Requirements

| ID      | Title                     | Requirement                                                                                                   | Category      | Priority | Status |
|---------|---------------------------|---------------------------------------------------------------------------------------------------------------|---------------|----------|--------|
| NFR-001 | Environment creation time | A new environment with up to 5 enabled components must reach `Synced` status within 5 minutes of the Git commit being pushed. | Performance   | High     | Open   |
| NFR-002 | Resource teardown time    | All resources for a disabled component must be removed within 3 minutes of the reconciliation cycle detecting the change. | Performance   | High     | Open   |
| NFR-003 | Portal availability       | The Backstage portal must be available 99.5% of the time during business hours (08:00–20:00 UTC, Mon–Fri).    | Availability  | High     | Open   |
| NFR-004 | Concurrent environments   | The platform must support at least 50 simultaneously active test environments without degraded reconciliation latency. | Scalability   | Medium   | Open   |
| NFR-005 | Namespace isolation       | Resources from one environment must not be reachable from another environment's pods via cluster-internal DNS or NetworkPolicy. | Security      | High     | Open   |
| NFR-006 | RBAC access control       | Only the environment owner and platform engineers may modify or delete an environment manifest; access must be enforced at the Git repository and ArgoCD level. | Security      | High     | Open   |
| NFR-007 | Manifest validation       | Invalid environment YAML (schema errors, unknown component types) must be rejected with a human-readable error message before any resources are created. | Reliability   | High     | Open   |
| NFR-008 | Idempotent provisioning   | Re-applying the same environment YAML must produce no changes when the cluster state already matches the desired state. | Reliability   | High     | Open   |
| NFR-009 | Audit trail               | All environment creation, modification, and deletion events must be traceable via Git commit history with author, timestamp, and diff. | Observability | Medium   | Open   |
| NFR-010 | Portal response time      | Backstage UI pages must load within 3 seconds under normal load (up to 20 concurrent users).                  | Performance   | Medium   | Open   |
| NFR-011 | docker-compose conversion accuracy | The docker-compose conversion tool must correctly map at least 90% of service definitions (image, ports, environment variables, volumes, dependencies) without manual correction. | Reliability   | High     | Open   |
| NFR-012 | Migration documentation coverage  | The migration runbook must cover 100% of application types currently running on the legacy platform before the legacy platform deprecation date. | Maintainability | High   | Open   |

---

## Constraints

| ID    | Title                       | Constraint                                                                                                   | Category  | Priority | Status |
|-------|-----------------------------|--------------------------------------------------------------------------------------------------------------|-----------|----------|--------|
| C-001 | GitOps tooling              | The reconciliation engine must be ArgoCD; no other CD tool may be introduced for environment management.      | Technical | High     | Open   |
| C-002 | Composition engine          | Crossplane must be used to abstract Kubernetes resource creation from the user-facing Environment claim down to workload resources; raw Kubernetes manifests must not be written directly by developers. | Technical | High     | Open   |
| C-003 | Developer portal            | Backstage must be used as the self-service UI and software catalog; no alternative portal may be deployed.    | Technical | High     | Open   |
| C-004 | Kubernetes runtime          | The platform must run on Kubernetes 1.26 or later.                                                           | Technical | High     | Open   |
| C-005 | GitOps source of truth      | All environment definitions must be stored in Git; direct `kubectl apply` by developers is not permitted.     | Technical | High     | Open   |
| C-006 | Namespace-per-environment   | Each environment must be provisioned in its own dedicated namespace; shared namespaces across environments are not allowed. | Technical | High     | Open   |
| C-007 | Environment YAML schema     | The environment manifest must be a valid Kubernetes Custom Resource claim backed by a Crossplane XRD so that it can be versioned and validated by the API server. | Technical | Medium   | Open   |
| C-008 | Supported database engines  | Initially, only PostgreSQL and Redis are supported as managed database components; other engines require a new Crossplane Composition. | Technical | Medium   | Open   |
| C-009 | No cloud provider dependency | The platform must run on any CNCF-conformant Kubernetes cluster without requiring managed cloud services (e.g., AWS RDS, GCP CloudSQL). | Technical | High     | Open   |
| C-010 | Legacy platform coexistence  | The legacy docker-compose platform must remain operational and unchanged during the migration period; the new IDP must not require its shutdown as a prerequisite. | Business  | High     | Open   |
| C-011 | Restricted legacy config access | Legacy environment configurations may not be fully accessible (restricted access); the migration tooling must handle partial or exported configs and document gaps for manual resolution. | Technical | High     | Open   |
| C-012 | Legacy platform deprecation  | The legacy docker-compose platform must be deprecated and decommissioned only after 100% of active environments have been successfully migrated and validated on the new platform. | Schedule  | High     | Open   |
