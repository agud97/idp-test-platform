# Use Case: Enable Application Component

## Overview

**Use Case ID:** UC-002
**Use Case Name:** Enable Application Component
**Primary Actor:** Developer
**Goal:** Provision all Kubernetes resources for an application component by setting its flag to `enabled: true`
**Status:** Draft

## Preconditions

- The environment manifest exists in the Git repository.
- The target component is currently set to `enabled: false` or is being added for the first time.
- The component type is registered in the platform catalog.

## Main Success Scenario

1. Developer opens the environment YAML manifest.
2. Developer sets `enabled: true` for the desired application component.
3. Developer commits and pushes the change to the Git repository.
4. ArgoCD detects the commit and triggers reconciliation.
5. System validates the updated manifest.
6. Crossplane creates the Kubernetes resources for the component (namespace entry, Deployment, Service, ConfigMap).
7. If the component type supports a database, system executes UC-004 (Provision Database Alongside App).
8. All resources reach a healthy running state.
9. ArgoCD updates environment status to `Synced`.

## Alternative Flows

### A1: Unknown Component Type

**Trigger:** The component type name does not exist in the platform catalog (step 5)
**Flow:**

1. System rejects the manifest with error: "Unknown component type: `<name>`".
2. ArgoCD sets environment status to `Degraded`.
3. Developer corrects the component type name or asks the platform engineer to register it (UC-010).
4. Use case continues at step 3.

### A2: Database Provisioning Failure

**Trigger:** UC-004 fails to provision the database (step 7)
**Flow:**

1. System marks the component as `Degraded`.
2. Developer views provisioning logs (UC-014) to identify the cause.
3. Platform engineer resolves the underlying issue.
4. Developer re-triggers reconciliation by pushing an empty commit.
5. Use case continues at step 4.

### A3: Resource Quota Exceeded

**Trigger:** Kubernetes namespace resource quota is exceeded during resource creation (step 6)
**Flow:**

1. System records a quota-exceeded event.
2. ArgoCD sets environment status to `Degraded`.
3. Developer adjusts replica count via UC-012 or disables another component first.
4. Use case continues at step 3.

## Postconditions

### Success Postconditions

- Component Deployment, Service, and ConfigMap exist in the environment namespace.
- If applicable, a DatabaseInstance record exists with status `Ready`.
- ArgoCD environment status is `Synced`.

### Failure Postconditions

- No partial resources are left in an unknown state (Crossplane rollback applies).
- ArgoCD environment status is `Degraded` with a descriptive error event.

## Business Rules

### BR-001: Atomic Component Provisioning

All resources for a component (Deployment, Service, ConfigMap, Database) are created together. Partial creation is not a valid end state.

### BR-002: Supported Database Engines

Only PostgreSQL and Redis are supported as managed databases. Other engines require a new Crossplane Composition registered by a platform engineer.
