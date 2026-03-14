# Use Case: View Environment Status

## Overview

**Use Case ID:** UC-007
**Use Case Name:** View Environment Status
**Primary Actor:** Developer, QA Engineer, Team Lead
**Goal:** Check the current provisioning status of a test environment to determine if it is ready to use
**Status:** Draft

## Preconditions

- Actor is authenticated in the Backstage portal.
- The environment exists in the software catalog.

## Main Success Scenario

1. Actor opens the environment detail page in the Backstage catalog (via UC-006 or a direct link).
2. System fetches the current reconciliation status from ArgoCD.
3. System displays the environment status: `Synced`, `Progressing`, or `Degraded`.
4. If status is `Synced`, system displays a list of healthy running components with their service endpoints.
5. If status is `Progressing`, system displays which resources are still being created or updated.
6. If status is `Degraded`, system displays a summary of the error and a link to provisioning logs (UC-014).

## Alternative Flows

### A1: Environment Not Found

**Trigger:** Environment has been deleted from the catalog or the actor lacks access (step 1)
**Flow:**

1. System displays "Environment not found or access denied".
2. Actor contacts the environment owner or platform engineer.
3. Use case ends.

### A2: ArgoCD Unreachable

**Trigger:** Backstage cannot fetch status from ArgoCD (step 2)
**Flow:**

1. System displays "Status unavailable — unable to reach reconciliation engine".
2. Platform engineer investigates ArgoCD availability.
3. Use case restarts at step 2 after ArgoCD is restored.

## Postconditions

### Success Postconditions

- Actor has seen the current environment status and can decide whether the environment is usable.
- No changes are made to the environment.

### Failure Postconditions

- Actor cannot determine environment status.
- Actor is shown a clear error message and directed to the appropriate contact.

## Business Rules

### BR-001: Status Values

Valid environment statuses are: `Synced` (ready), `Progressing` (being created/updated), `Degraded` (error).

### BR-002: Real-Time Data

The displayed status must reflect the current ArgoCD sync state, not a cached snapshot older than 30 seconds.
