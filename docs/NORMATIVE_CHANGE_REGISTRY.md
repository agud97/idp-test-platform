# Normative Change Registry

## Purpose

This register records the change history of the repository's normative documents:

- [requirements.md](requirements.md)
- [constraints.md](constraints.md)
- [acceptance_criteria.md](acceptance_criteria.md)
- [plan.yaml](plan.yaml)
- [use_cases.md](use_cases.md)
- [use_cases/](use_cases/)
- other specification-driving documents when they change platform behavior or implementation obligations

The goal is to make it easy to see:

- what changed
- when it changed
- which normative documents were affected
- why the change was made
- which commit is the audit source

## Source of Truth

The canonical history remains Git.

This file is a human-readable registry derived from Git history and should be updated whenever normative documents are changed intentionally.

## Change Log

| Date | Commit | Normative Documents Affected | Summary | Why |
|------|--------|-------------------------------|---------|-----|
| 2026-03-16 | `ce646b5` | `requirements.md`, `constraints.md`, `acceptance_criteria.md`, `plan.yaml`, `use_cases/UC-005-create-environment-via-portal.md`, `use_cases/UC-009-define-environment-template.md` | Aligned specs with the live Backstage service-selection UX: one unified `Service Selection` step, conditional service settings, and per-environment `account-api` overrides; corrected template-definition ACs to the Git-backed model. | The implemented Backstage flow had changed during the day and specs needed to reflect the actual runtime behavior without contradictions. |
| 2026-03-16 | `7cd753a` | `requirements.md`, `constraints.md`, `acceptance_criteria.md`, `plan.yaml`, `use_cases/UC-005-create-environment-via-portal.md`, `use_cases/UC-009-define-environment-template.md` | Switched normative model from image-baked scaffolder templates to live Git-backed template discovery. | Backstage template discovery was redesigned so template changes can be observed after Git commits without rebuilding the portal image. |
| 2026-03-15 | `c029f38` | `constraints.md`, `acceptance_criteria.md` | Documented that environment catalog entries must include `backstage.io/kubernetes-label-selector` or equivalent so the Kubernetes tab renders. | Live validation showed the Kubernetes tab stayed hidden without the selector annotation even when namespace and cluster annotations were present. |
| 2026-03-15 | `223df11` | `constraints.md` | Documented the Dockerfile JavaScript patching for OIDC sign-in as technical debt and captured upgrade risks. | The live Backstage image required compiled-JS patching rather than a clean source build, so the constraint set needed to describe that risk explicitly. |
| 2026-03-15 | `9e798e5` | `constraints.md`, `acceptance_criteria.md` | Updated the spec to require `spec.type: service` for environment catalog entries. | Live behavior showed the prebuilt Backstage image only renders the Kubernetes tab for `service` entities. |
| 2026-03-15 | `2aa1f8d` | `constraints.md`, `acceptance_criteria.md`, `requirements.md` | Added the `IdentityApi` implementation details needed for OIDC login compatibility, including `getProfileInfo()` and credentials shape. | OIDC login debugging found that the runtime patch must match the real Backstage frontend contract to avoid post-login failures. |
| 2026-03-15 | `2c8b4d8` | `constraints.md`, `acceptance_criteria.md`, `requirements.md`, selected use cases | Closed multiple specification gaps that had caused OIDC login failures in live validation. | The original specs were incomplete around the real Authentik and prebuilt-image behavior, so they were tightened to match the working solution. |
| 2026-03-15 | `d412053` | `constraints.md`, `acceptance_criteria.md`, `requirements.md`, selected use cases | Documented OIDC sign-in resolution behavior and guardrails needed to keep it working after redeploys. | The platform needed normative protection against regressions in Backstage authentication. |
| 2026-03-15 | `a318530` | `requirements.md`, `constraints.md`, `acceptance_criteria.md`, `plan.yaml`, use-case docs | Performed a broad alignment pass between implemented platform behavior and the normative spec set. | Implementation details discovered during validation had drifted from the original documents and needed reconciliation. |
| 2026-03-14 | `ded944c` | `requirements.md`, `constraints.md`, `acceptance_criteria.md`, `entity_model.md`, `use_cases.md`, `use_cases/`, `plan.yaml`, `vision.md` | Initial normative document set added for the project foundation. | This established the baseline specification and delivery plan for the repository. |

## Related Non-Normative Records

These documents are useful context, but they are not themselves the normative source:

- [POST_IMPLEMENTATION_DOC_CHANGES.md](POST_IMPLEMENTATION_DOC_CHANGES.md)
- [BACKSTAGE_OPERATIONS_GUIDE.md](BACKSTAGE_OPERATIONS_GUIDE.md)
- [ARCHITECTURE_OVERVIEW.md](ARCHITECTURE_OVERVIEW.md)
- [NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md](NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md)
- [LIVE_BACKSTAGE_GIT_TEMPLATE_ROLLOUT_REPORT.md](../LIVE_BACKSTAGE_GIT_TEMPLATE_ROLLOUT_REPORT.md)

## Update Rule

Whenever a commit changes normative intent, constraints, required behavior, or task obligations in the documents listed above:

1. add a new row to this register
2. record the date and commit hash
3. list the affected normative files
4. summarize what changed in one sentence
5. state why the change was necessary

Do not add rows for pure typo fixes unless they change meaning.

## Entry Template

Use this row template for future updates:

```markdown
| YYYY-MM-DD | `commit` | `requirements.md`, `constraints.md`, ... | One-sentence summary of the normative change. | One-sentence reason the change was necessary. |
```

## Writing Style

- Use one row per logical normative change, not per file.
- Prefer the commit that introduced the final accepted wording.
- List only the normative files whose meaning changed.
- In `Summary`, describe the behavioral or governance change, not the implementation detail.
- In `Why`, explain the trigger: contradiction, live validation finding, design decision, or scope change.
- If several commits are part of one change series, record the commit that best represents the final normative state and mention the broader series in `Summary` if needed.
