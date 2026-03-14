# Use Case: Track Migration Status Per Team

## Overview

**Use Case ID:** UC-019
**Use Case Name:** Track Migration Status Per Team
**Primary Actor:** Platform Engineer
**Goal:** Monitor which teams have migrated their environments to the new IDP platform in order to plan the deprecation of the legacy docker-compose platform
**Status:** Draft

## Preconditions

- Legacy environments have been registered in the migration tracker (LegacyEnvironment records exist).
- Platform engineer is authenticated in the Backstage portal or migration dashboard.

## Main Success Scenario

1. Platform engineer opens the migration tracking dashboard.
2. System displays a summary table: one row per team, showing total legacy environments, number migrated, number validated, number blocked, and overall migration percentage.
3. Platform engineer filters by team or migration status to focus on specific groups.
4. Platform engineer selects a team to view the detail view: individual legacy environment records with their `migration_status` values.
5. Platform engineer identifies teams or environments that are `blocked` or `pending` and need follow-up.
6. Platform engineer contacts the relevant team lead to unblock progress.
7. Platform engineer uses the summary to decide whether overall migration progress meets the deprecation readiness threshold.

## Alternative Flows

### A1: No Legacy Environments Registered

**Trigger:** A team has no LegacyEnvironment records in the tracker (step 2)
**Flow:**

1. System shows zero entries for that team.
2. Platform engineer contacts the team lead to register their legacy environments.
3. Team lead or platform engineer creates LegacyEnvironment records.
4. Use case continues at step 2.

### A2: Migration Status Data Stale

**Trigger:** LegacyEnvironment records have not been updated by developers after completing validation (step 4)
**Flow:**

1. Platform engineer notices environments that have been `in_progress` for an unusually long time.
2. Platform engineer contacts the developer to confirm actual status and update the record.
3. Developer updates LegacyEnvironment `migration_status`.
4. Use case continues at step 2.

## Postconditions

### Success Postconditions

- Platform engineer has an accurate, up-to-date view of migration progress across all teams.
- Blocked environments are identified and actioned.

### Failure Postconditions

- Dashboard data is incomplete due to missing or stale LegacyEnvironment records.
- Platform engineer manually contacts teams to collect status updates.

## Business Rules

### BR-001: Deprecation Gate

The legacy platform may only be deprecated and decommissioned after 100% of active LegacyEnvironment records have `migration_status: completed`. No exceptions.

### BR-002: All Environments Must Be Registered

Every environment running on the legacy platform must have a corresponding LegacyEnvironment record before migration tracking begins. Unregistered environments cannot be migrated.

### BR-003: Status Accuracy

Developers are responsible for keeping their LegacyEnvironment `migration_status` up to date. The platform engineer may override the status if a developer is unresponsive for more than 5 business days.
