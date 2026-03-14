# AGENTS.md — Execution Protocol

## Overview

This repository uses spec-driven development. Features are implemented by following
pre-approved plans derived from specifications.

**Your role:** Execute tasks from the plan sequentially, validate each task, and
report progress at checkpoints.

---

## Directory Structure

```
docs/                            # READ-ONLY — Do not modify during implementation
  requirements.md                # Functional requirements, NFRs, constraints (FR/NFR/C IDs)
  acceptance_criteria.md         # Behavioral specs in WHEN-THEN-SHALL format (AC IDs)
  constraints.md                 # Architecture rules, patterns, technical decisions
  entity_model.md                # Data model: ER diagram + attribute tables
  vision.md                      # Project vision, goals, technology stack
  use_cases.md                   # Use case overview (Mermaid diagram + traceability table)
  use_cases.puml                 # PlantUML use case diagram
  use_cases/                     # Individual use case specs (UC-001 … UC-022)
    UC-001-submit-environment-manifest.md
    UC-002-enable-application-component.md
    UC-003-disable-application-component.md
    UC-004-provision-database.md
    UC-005-create-environment-via-portal.md
    UC-006-browse-environment-catalog.md
    UC-007-view-environment-status.md
    UC-008-delete-environment.md
    UC-009-define-environment-template.md
    UC-010-register-component-type.md
    UC-011-isolate-environment-namespace.md
    UC-012-override-component-configuration.md
    UC-013-configure-gitops-reconciliation.md
    UC-014-view-provisioning-logs.md
    UC-015-convert-docker-compose.md
    UC-016-export-legacy-config.md
    UC-017-validate-migrated-environment.md
    UC-018-follow-migration-runbook.md
    UC-019-track-migration-status.md
    UC-020-register-legacy-environment.md
    UC-021-mark-migration-complete.md
    UC-022-initiate-deprecation.md
  plan.yaml                      # Execution plan: 5 phases, 30 tasks with dependencies
  spec-review.md                 # Spec review findings (18 issues, all resolved)
  spec-review-changes.md         # Change report: before/why/what/after for each fix

docs/status.md                   # Progress tracking — YOU create and update this file
```

Implementation code lives outside `docs/` in the repository root alongside
`platform/`, `cli/`, `environments/`, `catalog/`, `templates/`, `.github/`.

---

## Task List Workflow

### Initialization

When starting work on a feature:

```
1. READ docs/constraints.md
   → Understand architecture and constraints

2. READ docs/acceptance_criteria.md
   → Understand expected behavior (WHEN-THEN-SHALL format)

3. READ docs/plan.yaml
   → Understand phases, dependencies, checkpoints

4. READ docs/status.md
   → Determine current position in plan
   → If file doesn't exist, create it with phase-1, task-1.1 as NOT_STARTED

5. SET current_task = first task with status NOT_STARTED
```

### Main Execution Loop

```
WHILE current_task EXISTS:

    1. LOAD task card from docs/plan.yaml

    2. CHECK prerequisites
       FOR each task_id in task.depends_on:
           IF status[task_id] != COMPLETE:
               HALT with error "Dependency {task_id} not complete"

    3. UPDATE docs/status.md
       SET current_task.status = IN_PROGRESS

    4. EXECUTE task
       - Follow instructions in task card
       - Create/modify only files listed in task card
       - Apply constraints from docs/constraints.md

    5. VALIDATE task
       FOR each criterion in task.validation:
           RUN validation check
           IF check FAILS:
               INCREMENT retry_count
               IF retry_count > 2:
                   SET current_task.status = BLOCKED
                   HALT with VALIDATION_FAILURE report
               ELSE:
                   GOTO step 4 (retry execution)

    6. COMPLETE task
       UPDATE docs/status.md:
           SET current_task.status = COMPLETE
           ADD to completed_tasks table

    7. DETERMINE next action
       IF current_task is last task in phase:
           HALT with CHECKPOINT report
           WAIT for approval
           IF approved:
               SET current_task = first task of next phase
           ELSE IF revision requested:
               PROCESS revision instructions
               CONTINUE
       ELSE:
           SET current_task = next task in phase (task-{phase}.{n+1})

END WHILE

GENERATE completion report
```

### Phase Transitions

```
WHEN last task of phase completes:

    1. COLLECT phase summary
       - List all completed tasks
       - List all created files
       - Map to acceptance criteria from docs/acceptance_criteria.md

    2. GENERATE checkpoint report (see format below)

    3. HALT execution

    4. WAIT for human response:

       IF "APPROVED":
           CONTINUE to next phase

       IF "APPROVED WITH NOTES: {notes}":
           RECORD notes in docs/status.md
           CONTINUE to next phase

       IF "REVISE: task-{phase}.{n} - {instructions}":
           SET current_task = specified task
           SET current_task.status = IN_PROGRESS
           APPLY revision instructions
           RE-VALIDATE task
           RESUME from step 7 of main loop

       IF "BLOCKED: {reason}":
           RECORD blocker in docs/status.md
           HALT until blocker resolved
```

---

## Status File Format

Maintain `docs/status.md`:

```markdown
# Implementation Status: IDP Test Environment Platform

## Current Position
- Phase: {phase-id}
- Task: task-{phase}.{n}
- Status: NOT_STARTED | IN_PROGRESS | BLOCKED | COMPLETE

## Progress

### Phase 1: Project Foundation
| Task     | Status        | Notes |
|----------|---------------|-------|
| task-1.1 | · NOT_STARTED | |
| task-1.2 | · NOT_STARTED | |
| task-1.3 | · NOT_STARTED | |
| task-1.4 | · NOT_STARTED | |
| task-1.5 | · NOT_STARTED | |

### Phase 2: Platform Infrastructure
| Task     | Status        | Notes |
|----------|---------------|-------|
| task-2.1 | · NOT_STARTED | |
| task-2.2 | · NOT_STARTED | |
| task-2.3 | · NOT_STARTED | |
| task-2.4 | · NOT_STARTED | |
| task-2.5 | · NOT_STARTED | |
| task-2.6 | · NOT_STARTED | |

### Phase 3: CLI Migration Tool
| Task     | Status        | Notes |
|----------|---------------|-------|
| task-3.1 | · NOT_STARTED | |
| task-3.2 | · NOT_STARTED | |
| task-3.3 | · NOT_STARTED | |
| task-3.4 | · NOT_STARTED | |
| task-3.5 | · NOT_STARTED | |
| task-3.6 | · NOT_STARTED | |

### Phase 4: Backstage Portal
| Task     | Status        | Notes |
|----------|---------------|-------|
| task-4.1 | · NOT_STARTED | |
| task-4.2 | · NOT_STARTED | |
| task-4.3 | · NOT_STARTED | |
| task-4.4 | · NOT_STARTED | |
| task-4.5 | · NOT_STARTED | |
| task-4.6 | · NOT_STARTED | |

### Phase 5: Acceptance Testing & Hardening
| Task     | Status        | Notes |
|----------|---------------|-------|
| task-5.1 | · NOT_STARTED | |
| task-5.2 | · NOT_STARTED | |
| task-5.3 | · NOT_STARTED | |
| task-5.4 | · NOT_STARTED | |
| task-5.5 | · NOT_STARTED | |
| task-5.6 | · NOT_STARTED | |

## Checkpoints
| Phase   | Status      | Approved |
|---------|-------------|----------|
| phase-1 | NOT_REACHED | |
| phase-2 | NOT_REACHED | |
| phase-3 | NOT_REACHED | |
| phase-4 | NOT_REACHED | |
| phase-5 | NOT_REACHED | |

## Blockers
<!-- empty if none -->

## Deviations
<!-- record any approved deviations from plan -->
```

IMPORTANT: Add notes to the tasks upon completion.

---

## Checkpoint Report Format

```markdown
# Checkpoint: Phase {n} Complete

## Phase Summary
- Phase: {phase-id} — {phase name}
- Tasks completed: {n}/{n}

## Completed Tasks
| Task     | Description |
|----------|-------------|
| task-{phase}.1 | {task name} |
| task-{phase}.2 | {task name} |

## Artifacts Created
| File | Purpose |
|------|---------|
| {path} | {one-line description} |

## Acceptance Criteria Coverage
| AC ID  | Description | Status | Implemented In |
|--------|-------------|--------|----------------|
| AC-001 | {name}      | ✓      | {file or test} |

## Validation Results
- Compilation: PASS | FAIL
- Tests: {n}/{n} passing
- Constraints: {n}/{n} satisfied

## Issues / Decisions
{list any ambiguities resolved, minor deviations, or concerns}

---

**CHECKPOINT REACHED — AWAITING APPROVAL**

Respond with:
- `APPROVED` — proceed to phase {n+1}
- `APPROVED WITH NOTES: {notes}` — proceed with adjustments
- `REVISE: task-{phase}.{n} - {instructions}` — fix before proceeding
- `BLOCKED: {reason}` — stop for discussion
```

---

## Blocker Report Format

```markdown
# BLOCKED: task-{phase}.{n}

## Type
SPEC_AMBIGUITY | SPEC_CONFLICT | TECHNICAL | VALIDATION_FAILURE | DEPENDENCY

## Summary
{one sentence description}

## Details
{full explanation of the problem}

## Evidence
{code snippets, error messages, spec quotes as relevant}

## Attempted Solutions
1. {what you tried} → {result}
2. {what you tried} → {result}

## Options
1. {option}
   - Impact: {on plan, timeline, architecture}
   - Tradeoff: {pros and cons}

2. {option}
   - Impact: {on plan, timeline, architecture}
   - Tradeoff: {pros and cons}

## Recommendation
{which option you suggest and why}

---

**BLOCKED — AWAITING RESOLUTION**

Respond with:
- `PROCEED WITH: {option number}` — continue with specified approach
- `PROCEED WITH: {custom instructions}` — continue with your guidance
- `ABORT TASK` — skip this task, adjust plan
```

---

## Rules

### MUST
- Process tasks in order within each phase
- Verify all prerequisites before starting a task
- Validate all acceptance criteria before marking complete
- Update `docs/status.md` after each task state change
- Stop at every checkpoint and wait for approval
- Stop immediately when blocked

### MUST NOT
- Modify files in `docs/` directory (except `docs/status.md`)
- Skip tasks or reorder tasks within a phase
- Proceed past checkpoint without explicit approval
- Proceed when blocked without resolution
- Create files not specified in the current task card
- Modify files outside current task's scope

### SHOULD
- Reference constraint IDs (e.g. §2.1, BLK-001) when applying rules from `docs/constraints.md`
- Reference AC IDs when validating behavior against `docs/acceptance_criteria.md`
- Record reasoning for non-obvious implementation decisions in `docs/status.md`
- Note potential improvements discovered during implementation

---

## Quick Reference

| State | Action |
|-------|--------|
| Starting feature | Initialize → Read specs → Read plan → Find current task |
| Starting task | Load card → Check prerequisites → Update status → Execute |
| Task complete | Validate → Update status → Check if phase end |
| Phase complete | Generate checkpoint report → HALT → Wait for approval |
| Blocked | Generate blocker report → HALT → Wait for resolution |
| All phases complete | Generate completion report |

---

## Task Index

| Task     | Phase | Description |
|----------|-------|-------------|
| task-1.1 | 1 — Project Foundation | Scaffold repository structure |
| task-1.2 | 1 — Project Foundation | Define Environment CRD |
| task-1.3 | 1 — Project Foundation | Initialise Go CLI module |
| task-1.4 | 1 — Project Foundation | Create CLI test fixtures |
| task-1.5 | 1 — Project Foundation | Create CI pipeline skeleton |
| task-2.1 | 2 — Platform Infrastructure | Crossplane XRD + Composition: postgresql |
| task-2.2 | 2 — Platform Infrastructure | Crossplane XRD + Composition: redis |
| task-2.3 | 2 — Platform Infrastructure | Crossplane XRD + Composition: webapp |
| task-2.4 | 2 — Platform Infrastructure | Crossplane EnvironmentConfig for cluster defaults |
| task-2.5 | 2 — Platform Infrastructure | ArgoCD ApplicationSet + platform Application manifests |
| task-2.6 | 2 — Platform Infrastructure | Baseline NetworkPolicy templates |
| task-3.1 | 3 — CLI Migration Tool | Implement pkg/converter |
| task-3.2 | 3 — CLI Migration Tool | Implement pkg/exporter |
| task-3.3 | 3 — CLI Migration Tool | Implement pkg/validator |
| task-3.4 | 3 — CLI Migration Tool | Implement CLI commands: migrate, export, validate |
| task-3.5 | 3 — CLI Migration Tool | Implement CLI commands: register-legacy, complete-migration, deprecate-legacy |
| task-3.6 | 3 — CLI Migration Tool | CLI integration tests |
| task-4.1 | 4 — Backstage Portal | Backstage authentication: Authentik OIDC |
| task-4.2 | 4 — Backstage Portal | Backstage catalog and Kubernetes plugin |
| task-4.3 | 4 — Backstage Portal | Backstage scaffolder template: New Environment |
| task-4.4 | 4 — Backstage Portal | Migration dashboard: backend plugin |
| task-4.5 | 4 — Backstage Portal | Migration dashboard: frontend plugin |
| task-4.6 | 4 — Backstage Portal | TechDocs migration runbooks |
| task-5.1 | 5 — Acceptance Testing & Hardening | Acceptance tests: environment lifecycle |
| task-5.2 | 5 — Acceptance Testing & Hardening | Acceptance tests: component overrides and database |
| task-5.3 | 5 — Acceptance Testing & Hardening | Acceptance tests: GitOps self-heal and isolation |
| task-5.4 | 5 — Acceptance Testing & Hardening | Acceptance tests: migration CLI |
| task-5.5 | 5 — Acceptance Testing & Hardening | Security scan and credential audit |
| task-5.6 | 5 — Acceptance Testing & Hardening | Load test: 50 concurrent environments |
