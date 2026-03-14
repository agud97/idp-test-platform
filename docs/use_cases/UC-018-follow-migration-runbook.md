# Use Case: Follow Migration Runbook

## Overview

**Use Case ID:** UC-018
**Use Case Name:** Follow Migration Runbook
**Primary Actor:** Developer
**Goal:** Migrate a legacy docker-compose application to the IDP platform independently by following a step-by-step runbook, without requiring platform engineer involvement
**Status:** Draft

## Preconditions

- A migration runbook exists for the developer's application type.
- Developer has access to the runbook documentation (Backstage TechDocs or equivalent).
- Developer has completed UC-016 (Export Legacy Config) or has direct access to the docker-compose file.

## Main Success Scenario

1. Developer identifies their application type (e.g., "web app + PostgreSQL + Redis").
2. Developer opens the migration runbook for that application type in the documentation portal.
3. Developer follows step 1 of the runbook: export legacy config (UC-016) or confirm the docker-compose file is available.
4. Developer follows step 2: convert the config to an IDP manifest (UC-015).
5. Developer follows step 3: review and fix any items flagged in the conversion report.
6. Developer follows step 4: submit the manifest to Git and provision the IDP environment (UC-001).
7. Developer follows step 5: validate the migrated environment against the legacy (UC-017).
8. Developer marks the runbook as completed and updates LegacyEnvironment `migration_status` to `validated`.

## Alternative Flows

### A1: Application Type Not Covered by Runbook

**Trigger:** No runbook exists for the developer's specific application type (step 2)
**Flow:**

1. Developer reports the missing runbook to the platform engineer.
2. Platform engineer creates the runbook for the application type (NFR-012 requires 100% coverage).
3. Developer is notified when the runbook is published.
4. Use case restarts at step 2.

### A2: Runbook Step Fails

**Trigger:** Developer encounters an error that the runbook does not address (any step)
**Flow:**

1. Developer documents the failure: step number, error message, environment details.
2. Developer sets LegacyEnvironment `migration_status` to `blocked`.
3. Developer escalates to the platform engineer with the documented failure.
4. Platform engineer resolves the issue and updates the runbook to cover the edge case.
5. Developer resumes the runbook from the failed step.
6. Use case continues at the appropriate step.

### A3: Runbook Is Outdated

**Trigger:** Developer notices that runbook instructions do not match the current platform version (any step)
**Flow:**

1. Developer flags the runbook as potentially outdated in the documentation portal.
2. Platform engineer reviews and updates the runbook.
3. Developer uses the updated runbook.
4. Use case continues at the affected step.

## Postconditions

### Success Postconditions

- IDP environment is provisioned and validated as a replacement for the legacy environment.
- LegacyEnvironment `migration_status` is `validated`.
- Developer completed the migration independently without escalation to the platform engineer.

### Failure Postconditions

- Migration is paused at a specific runbook step.
- LegacyEnvironment `migration_status` is `blocked`.
- Platform engineer is engaged to resolve the blocker.

## Business Rules

### BR-001: Full Coverage Requirement

The migration runbook must cover 100% of application types currently running on the legacy platform before the legacy platform deprecation date.

### BR-002: Self-Service Goal

The runbook must be detailed enough for a developer with no Kubernetes knowledge to complete the migration without platform engineer assistance for the standard (happy path) scenario.

### BR-003: Runbook Versioning

Runbooks must be versioned and linked to the IDP platform version they apply to. Developers must use the runbook version matching the current platform release.
