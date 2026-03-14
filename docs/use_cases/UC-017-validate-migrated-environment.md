# Use Case: Validate Migrated Environment

## Overview

**Use Case ID:** UC-017
**Use Case Name:** Validate Migrated Environment
**Primary Actor:** Developer, QA Engineer
**Goal:** Verify functional equivalence between the migrated IDP environment and the legacy docker-compose environment before switching over permanently
**Status:** Draft

## Preconditions

- The migrated IDP environment is provisioned and has status `Synced` (UC-001 completed).
- The legacy docker-compose environment is still running and accessible.
- Validation test suite or checklist exists for the application stack.

## Main Success Scenario

1. Developer/QA Engineer opens the IDP environment and the legacy environment side by side.
2. Actor runs the validation test suite against both environments.
3. System records test results for each environment.
4. Actor compares outputs: API responses, service availability, database connectivity, and configuration values.
5. All test cases pass on the IDP environment with results equivalent to the legacy environment.
6. Actor marks the IDP environment as "validated" in the migration tracker.
7. Actor updates the LegacyEnvironment record: sets `migration_status` to `validated`.

## Alternative Flows

### A1: Functional Discrepancy Found

**Trigger:** One or more test cases produce different results between legacy and IDP environments (step 4)
**Flow:**

1. Actor documents the discrepancy (affected service, expected vs actual behaviour).
2. Developer investigates the root cause: missing environment variable, incorrect image tag, misconfigured override.
3. Developer corrects the IDP environment manifest and pushes the fix.
4. ArgoCD reconciles the change.
5. Use case restarts at step 2.

### A2: Legacy Environment Unreachable During Validation

**Trigger:** The legacy environment is not accessible at validation time (step 1)
**Flow:**

1. Actor records that legacy comparison was not possible.
2. Actor performs standalone validation against the IDP environment using known expected values.
3. Actor notes the limitation in the validation report.
4. Use case continues at step 5 if standalone validation passes.

### A3: IDP Environment Degraded During Validation

**Trigger:** The IDP environment enters `Degraded` status during the validation run (step 2)
**Flow:**

1. Actor pauses validation.
2. Developer resolves the degraded state (see UC-014, UC-002 as appropriate).
3. Use case restarts at step 2.

## Postconditions

### Success Postconditions

- LegacyEnvironment `migration_status` is set to `validated`.
- Validation report is saved and linked to the LegacyEnvironment record.
- Team lead and platform engineer are notified that the environment is ready for cutover.

### Failure Postconditions

- LegacyEnvironment `migration_status` remains `in_progress`.
- Discrepancies are documented for developer resolution.
- Legacy environment continues to run until validation passes.

## Business Rules

### BR-001: Parallel Operation Required

Both the legacy and IDP environments must be running simultaneously during validation. The legacy platform must not be shut down before validation is complete.

### BR-002: Validation Before Cutover

An environment's `migration_status` must be `validated` before the legacy environment can be decommissioned.

### BR-003: Written Validation Report

Every validation run must produce a written report (pass/fail per test case) stored with the LegacyEnvironment record.
