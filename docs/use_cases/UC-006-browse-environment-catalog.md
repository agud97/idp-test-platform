# Use Case: Browse Environment Catalog

## Overview

**Use Case ID:** UC-006
**Use Case Name:** Browse Environment Catalog
**Primary Actor:** Developer, QA Engineer, Team Lead
**Goal:** View and search all active test environments in the Backstage software catalog
**Status:** Draft

## Preconditions

- Actor is authenticated in the Backstage portal.
- The Backstage software catalog is operational.

## Main Success Scenario

1. Actor opens the Backstage portal and navigates to the "Catalog" section.
2. System displays a list of all environments the actor has access to, with basic metadata (name, owner, team, status).
3. Actor optionally filters the list by team, owner, or status.
4. Actor selects an environment from the list.
5. System displays the environment detail page showing: owner, team, component list, current status, and links to provisioning logs.

## Alternative Flows

### A1: No Environments Found

**Trigger:** No environments match the applied filter or none exist (step 2 or step 3)
**Flow:**

1. System displays "No environments found" message.
2. Actor clears filters or creates a new environment (UC-005).
3. Use case ends.

### A2: Portal Unavailable

**Trigger:** Backstage portal is not accessible (step 1)
**Flow:**

1. Browser displays a connection error.
2. Actor waits and retries, or contacts the platform engineer.
3. Use case restarts at step 1 after the portal is restored.

## Postconditions

### Success Postconditions

- Actor has viewed the list of environments and/or an environment detail page.
- No changes are made to any environment.

### Failure Postconditions

- Actor cannot access the catalog.
- Actor is informed of the portal unavailability.

## Business Rules

### BR-001: Access Scope

Developers and QA engineers see environments they own or that belong to their team. Platform engineers see all environments.

### BR-002: Read-Only View

Browsing the catalog does not modify any environment state.
