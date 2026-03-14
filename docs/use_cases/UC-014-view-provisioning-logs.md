# Use Case: View Provisioning Logs

## Overview

**Use Case ID:** UC-014
**Use Case Name:** View Provisioning Logs
**Primary Actor:** Developer
**Goal:** Inspect provisioning events and error messages in the Backstage portal to diagnose environment failures without direct Kubernetes access
**Status:** Draft

## Preconditions

- Developer is authenticated in the Backstage portal.
- The environment exists in the Backstage catalog.
- The environment has at least one recorded ArgoCD event or reconciliation history entry.

## Main Success Scenario

1. Developer opens the environment detail page in the Backstage catalog.
2. Developer navigates to the "Provisioning Logs" or "Events" tab.
3. System fetches recent reconciliation events from ArgoCD for this environment.
4. System displays a chronological list of events: resource creation, update, deletion, and error messages.
5. Developer identifies the event or error relevant to the issue.
6. If the error message is insufficient, developer expands an event to see the full error detail and affected resource name.

## Alternative Flows

### A1: No Events Available

**Trigger:** The environment has no recorded events (e.g., it was just created and reconciliation has not started) (step 4)
**Flow:**

1. System displays "No provisioning events found yet".
2. Developer waits for the first reconciliation cycle and refreshes.
3. Use case continues at step 4.

### A2: ArgoCD Unreachable

**Trigger:** Backstage cannot fetch events from ArgoCD (step 3)
**Flow:**

1. System displays "Unable to retrieve logs — reconciliation engine unavailable".
2. Developer contacts the platform engineer.
3. Use case ends.

### A3: Access Denied

**Trigger:** Developer tries to view logs for an environment they do not own (step 2)
**Flow:**

1. System displays "Access denied — you are not the owner of this environment".
2. Developer requests access from the owner or platform engineer.
3. Use case ends.

## Postconditions

### Success Postconditions

- Developer has viewed provisioning events and identified the cause of any failure.
- No changes are made to the environment.

### Failure Postconditions

- Developer cannot view logs due to portal or ArgoCD unavailability.
- Developer is directed to the platform engineer for assistance.

## Business Rules

### BR-001: Access Scope

Only the environment owner and platform engineers can view provisioning logs for a given environment.

### BR-002: Retention Period

Provisioning events must be retained and accessible for at least 7 days after the event occurred.
