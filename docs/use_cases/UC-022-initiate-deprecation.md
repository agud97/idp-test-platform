# Use Case: Initiate Legacy Platform Deprecation

## Overview

**Use Case ID:** UC-022
**Use Case Name:** Initiate Legacy Platform Deprecation
**Primary Actor:** Platform Engineer
**Goal:** Trigger a deprecation gate check and, if all migrations are complete, produce a formal deprecation report authorising shutdown of the legacy docker-compose platform
**Status:** Draft

> **Resolves:** BLK-004 — defines the deprecation workflow

## Preconditions

- All legacy environments have been registered (UC-020).
- Platform engineer is authenticated.

## Main Success Scenario

1. Platform engineer runs: `idp deprecate-legacy --dry-run` to preview the gate status.
2. CLI queries all `LEGACY_ENVIRONMENT` records and counts environments by `migration_status`.
3. CLI displays a summary table: total, completed, validated, in_progress, pending, blocked.
4. If all environments have `migration_status: completed`, CLI displays: "Gate PASSED — all N environments completed."
5. Platform engineer runs: `idp deprecate-legacy --confirm` to generate the formal report.
6. CLI writes a deprecation report file `deprecation-report-<date>.md` listing every migrated environment, its completion timestamp, and the IDP environment it maps to.
7. CLI sets a platform-level flag `LEGACY_PLATFORM_DEPRECATED=true` in the tracker (a single-row config record).
8. Platform engineer uses the report as the authorisation artefact to physically decommission the legacy infrastructure.

## Alternative Flows

### A1: Gate Fails — Incomplete Environments Remain

**Trigger:** One or more environments have `migration_status != completed` (step 4)
**Flow:**

1. CLI displays: "Gate FAILED — deprecation blocked."
2. CLI lists each blocking environment: name, team, current status.
3. Platform engineer resolves each blocker (complete validations or escalate blocked environments).
4. Use case restarts at step 1.

### A2: No Environments Registered

**Trigger:** Zero `LEGACY_ENVIRONMENT` records exist (step 2)
**Flow:**

1. CLI exits with error: "No legacy environments registered. Register environments first with `idp register-legacy` before initiating deprecation."
2. Platform engineer registers all legacy environments (UC-020).
3. Use case restarts at step 1.

## Postconditions

### Success Postconditions

- A `deprecation-report-<date>.md` file is generated and saved locally.
- The tracker records `LEGACY_PLATFORM_DEPRECATED=true` with the timestamp.
- The platform engineer has a formal authorisation artefact to proceed with infrastructure decommissioning.

### Failure Postconditions

- No report is generated.
- No flag is set.
- Platform engineer has a list of blocking environments to resolve.

## Business Rules

### BR-001: 100% Completion Required

The deprecation command MUST be blocked unless 100% of registered `LEGACY_ENVIRONMENT` records have `migration_status: completed`. No exceptions.

### BR-002: Dry Run First

The `--dry-run` flag MUST be the default mode. Generating the formal report requires an explicit `--confirm` flag to prevent accidental execution.

### BR-003: Report as Audit Artefact

The deprecation report MUST include: generation timestamp, operator username, total environment count, list of all environments with their `completed_at` timestamps and mapped IDP environment IDs. The report MUST be committed to the GitOps repository under `docs/deprecation/` as the permanent audit trail.
