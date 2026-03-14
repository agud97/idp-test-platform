# Vision: IDP Platform for Test Environment Provisioning

## Problem Statement

Development and QA teams spend significant time manually creating and tearing down test environments.
Setting up namespaces, deployments, services, and databases requires deep Kubernetes knowledge and
is error-prone, slow, and inconsistent across teams.

## Vision Statement

An Internal Developer Platform (IDP) that enables any engineer to provision a fully functional,
isolated test environment by submitting a single YAML file — without Kubernetes expertise.

## Target Users

- **Developer** — wants to spin up a personal test environment quickly to validate a feature branch
- **QA Engineer** — needs reproducible, isolated environments for test suites
- **Team Lead / Architect** — defines environment templates and governs what components are available
- **Platform Engineer** — maintains the IDP infrastructure, tooling, and available component catalog

## Core Concept

The user submits one YAML manifest listing the desired applications for their environment.
Each application entry has an `enabled: true/false` flag:

- `enabled: true` — the platform provisions all required Kubernetes resources (namespace, deployments, services, databases, config)
- `enabled: false` — all resources for that application are removed automatically

The environment lifecycle is fully GitOps-driven: the YAML file is committed to a Git repository,
and the platform reconciles the live cluster state to match.

## Key Capabilities

1. **Single-file provisioning** — one YAML describes the full environment topology
2. **Component on/off switching** — toggle any application without editing Kubernetes manifests
3. **Full resource lifecycle** — create, update, and delete Kubernetes resources automatically
4. **Database provisioning** — databases (PostgreSQL, Redis, etc.) created alongside the application
5. **Self-service via Backstage** — developers use a UI portal to create environments from templates
6. **GitOps reconciliation** — ArgoCD watches the Git repo, applies Environment claims, and keeps the control-plane state in sync
7. **Infrastructure abstraction** — Crossplane Compositions translate Environment claims into namespaces, policies, and workload/database resources

## Technology Stack

| Tool       | Role                                                              |
|------------|-------------------------------------------------------------------|
| ArgoCD     | GitOps controller — reconciles cluster state from Git            |
| Crossplane | Composition engine — translates Environment claims into namespaces and K8s resources |
| Backstage  | Developer portal — self-service UI, software catalog, scaffolder |
| Git        | Source of truth for environment definitions                      |
| Kubernetes | Runtime platform                                                 |

## Success Criteria

- A developer with no Kubernetes knowledge can create a working environment in under 5 minutes
- Disabling an application removes all its resources within 3 minutes
- Environments are fully reproducible from the same YAML input
- Platform engineers can add new component types without changing the developer-facing YAML schema
