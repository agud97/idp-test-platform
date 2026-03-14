# Use Case: Mark Migration as Complete

## Overview

**Use Case ID:** UC-021
**Use Case Name:** Mark Migration as Complete
**Primary Actor:** Platform Engineer
**Goal:** Transition a legacy environment's status from `validated` to `completed` after its legacy instance has been decommissioned, unblocking the deprecation gate
**Status:** Draft

> **Resolves:** BLK-003 — defines the `validated → completed` transition

## Preconditions

- The LEGACY_ENVIRONMENT record exists with `migration_status: validated`.
- The corresponding IDP environment has been running stably for at least one full business day after validation.
- The legacy docker-compose environment has been shut down and is no longer serving traffic.

## Main Success Scenario

1. Platform engineer confirms with the owning team that they no longer need the legacy environment.
2. Platform engineer shuts down (stops and removes) the legacy docker-compose environment.
3. Platform engineer runs: `idp complete-migration --env-id <legacy-env-id>`.
4. CLI verifies that `migration_status` is currently `validated` (not `pending`, `in_progress`, or `blocked`).
5. CLI sets `migration_status: completed` and records `completed_at` timestamp.
6. CLI prints confirmation: "Migration of `<env-name>` marked as completed."
7. The team's derived `migrated` count in the dashboard increases by 1.

## Alternative Flows

### A1: Environment Not Yet Validated

**Trigger:** `migration_status` is not `validated` when the command runs (step 4)
**Flow:**

1. CLI exits with error: "Cannot mark as completed: current status is `<status>`. Environment must be `validated` first."
2. Platform engineer completes the validation step (UC-017) before retrying.
3. Use case ends.

### A2: Legacy Environment Still Running

**Trigger:** Platform engineer attempts to mark complete before shutting down the legacy environment (logical check only — CLI cannot verify this automatically)
**Flow:**

1. CLI proceeds with the status update (it cannot verify the legacy environment's runtime state).
2. Platform engineer is responsible for ensuring the legacy environment is actually stopped.
3. If discovered running later, the platform engineer reverts status to `validated` using `idp revert-migration`.

## Postconditions

### Success Postconditions

- `LEGACY_ENVIRONMENT.migration_status` = `completed`, `completed_at` = current timestamp.
- The deprecation gate (UC-022) counts this environment toward the 100% threshold.
- The legacy environment is no longer running.

### Failure Postconditions

- `migration_status` is unchanged.
- Platform engineer receives a clear error with the required corrective action.

## Business Rules

### BR-001: Sequential Lifecycle Only

The only valid transition into `completed` is from `validated`. Skipping validation to mark directly as `completed` is prohibited.

### BR-002: Irreversibility

Once `completed`, the status can only be changed back to `validated` by a platform engineer using an explicit revert command, which requires a written justification (logged as an audit event).
