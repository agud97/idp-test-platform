# Use Case: Register Legacy Environment

## Overview

**Use Case ID:** UC-020
**Use Case Name:** Register Legacy Environment
**Primary Actor:** Platform Engineer
**Goal:** Create a migration tracker record for a legacy docker-compose environment so that its progress can be monitored and included in the deprecation gate
**Status:** Draft

> **Resolves:** MAJ-002, BLK-003 (precondition for UC-021, UC-022)

## Preconditions

- Platform engineer is authenticated.
- The legacy environment is known by name and belongs to an identified team.
- A TEAM record already exists for the owning team.

## Main Success Scenario

1. Platform engineer runs: `idp register-legacy --name <env-name> --team <team-name> [--compose-path <path>]`.
2. CLI validates that the team exists in the tracker.
3. CLI creates a `LEGACY_ENVIRONMENT` record with `migration_status: pending`, associating it with the team.
4. CLI prints confirmation: "Registered legacy environment `<env-name>` for team `<team-name>` (ID: <id>)."
5. The environment is now visible in the migration tracking dashboard (UC-019) under the owning team.

## Alternative Flows

### A1: Team Does Not Exist

**Trigger:** The specified team name has no TEAM record (step 2)
**Flow:**

1. CLI exits with error: "Team `<name>` not found. Create the team first with `idp register-team`."
2. Platform engineer creates the team record and re-runs the command.
3. Use case restarts at step 1.

### A2: Duplicate Registration

**Trigger:** A LEGACY_ENVIRONMENT record with the same name and team already exists (step 3)
**Flow:**

1. CLI exits with error: "Legacy environment `<name>` is already registered for team `<team>`."
2. Platform engineer verifies the existing record and updates it if needed using `idp update-legacy`.
3. Use case ends.

## Postconditions

### Success Postconditions

- A `LEGACY_ENVIRONMENT` record exists with `migration_status: pending` and the correct `team_id`.
- The environment appears in the dashboard total count for the team.

### Failure Postconditions

- No record is created.
- Platform engineer receives a clear error message with the required corrective action.

## Business Rules

### BR-001: All Environments Must Be Registered Before Migration Starts

Every environment running on the legacy platform must be registered before the migration tracking dashboard is considered authoritative. Unregistered environments are invisible to the deprecation gate.

### BR-002: compose_file_path Is Optional

The compose file path may not be available due to access restrictions (C-011). Registration proceeds without it; the path can be updated later when available.
