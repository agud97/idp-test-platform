# Session Bootstrap

## Purpose

Read this file at the start of any new session before doing project work.

It exists to minimize repeated rediscovery and to point a new agent to the
smallest set of files that capture the real state of the project.

## Repository

- Repo path: `/root/codex/idp-test`
- Git remote: `https://github.com/agud97/idp-test-platform.git`
- Working branch used during implementation: `feature/phase-1-foundation`

## Current Project State

- Implementation is complete through phase 5.
- Backstage, ArgoCD, Crossplane, CLI, acceptance, security, and load work were all implemented and validated.
- The repository includes post-implementation doc alignment so the current `docs/*` should be treated as the authoritative plan/spec baseline.

## Read These First

Minimum required reading order:

1. [README.md](README.md)
2. [INDEX.md](INDEX.md)
3. [docs/status.md](docs/status.md)
4. [docs/POST_IMPLEMENTATION_DOC_CHANGES.md](docs/POST_IMPLEMENTATION_DOC_CHANGES.md)
5. [docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md](docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md)

Then, depending on task type:

- operational/runtime work:
  - [HANDOFF.md](HANDOFF.md)
  - [OPERATIONS_CHECKLIST.md](OPERATIONS_CHECKLIST.md)

- architecture or implementation changes:
  - [docs/ARCHITECTURE_OVERVIEW.md](docs/ARCHITECTURE_OVERVIEW.md)
  - [docs/constraints.md](docs/constraints.md)
  - [docs/plan.yaml](docs/plan.yaml)

- testing or migration usage:
  - [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md)
  - [docs/USER_JOURNEYS.md](docs/USER_JOURNEYS.md)
  - [docs/FAQ.md](docs/FAQ.md)

## Critical Project Facts

### 1. GitOps ownership model

Do not assume ArgoCD directly owns workload `Deployment` objects.

In this project:
- ArgoCD owns the Git-managed `Environment` claim layer
- Crossplane owns the runtime resource layer

This matters for:
- self-heal expectations
- drift tests
- troubleshooting

### 2. Backstage runtime model

Do not assume this repo contains a normal live Backstage source monorepo.

In this project:
- Backstage runs from a prebuilt image
- runtime wiring lives under `platform/backstage/`
- source artifacts under `packages/backend/...` and `packages/app/...` exist for spec traceability, but live behavior is wired through the image/runtime path

### 3. Phase 2.7 is mandatory context

Do not treat `phase-2.7` as optional history.

It closes the required provisioning path:
- `Environment` claim/XRD
- `Environment` Composition
- provider wiring
- Crossplane Functions
- repo-hosted Crossplane GitOps deployment

Without this context, phase 5 will be misread.

### 4. AC-003 semantics were corrected

Do not assume invalid manifests surface as Argo health `Degraded`.

Authoritative current meaning:
- invalid manifest
- zero resources created
- human-readable Argo sync/operation failure

See:
- [docs/acceptance_criteria.md](docs/acceptance_criteria.md)
- [docs/POST_IMPLEMENTATION_DOC_CHANGES.md](docs/POST_IMPLEMENTATION_DOC_CHANGES.md)

### 5. Backstage is useful, but not the source of truth

Use Backstage for:
- self-service creation
- catalog browsing
- migration dashboard
- runbooks/docs

Use Git as the source of truth for:
- environment manifests
- bulk updates
- reviewable config changes

Use `kubectl` for:
- exact runtime diagnosis

## Environment Assumptions Used In Validation

- Kubeconfig path used during validation:
  - `/root/codex/kubeconfig_6144665`
- Backstage local access during validation:
  - `http://localhost:7007`
- Git branch used by GitOps and scaffolder flows:
  - `feature/phase-1-foundation`

## Known External Risks

These are not repo-memory problems; they are live-environment risks:

- Kubernetes API instability
- GitHub token/push access issues
- registry/image pull failures
- Authentik/OIDC availability
- cluster capacity limits for load testing
- CNI / DNS / NetworkPolicy behavior

See:
- [docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md](docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md)

## Recommended Session Opening Prompt

Use something like this at the start of a new session:

```text
Read README.md, INDEX.md, docs/status.md, docs/POST_IMPLEMENTATION_DOC_CHANGES.md, and docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md first. Treat them as mandatory context for /root/codex/idp-test before doing any work.
```

## If You Need One More File

If only one additional file is read after the minimum set, make it:

- [HANDOFF.md](HANDOFF.md)

It gives the best practical bridge from documentation to real operation.
