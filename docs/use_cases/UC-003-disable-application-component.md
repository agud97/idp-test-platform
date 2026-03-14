# Use Case: Disable Application Component

## Overview

**Use Case ID:** UC-003
**Use Case Name:** Disable Application Component
**Primary Actor:** Developer
**Goal:** Remove all Kubernetes resources for an application component by setting its flag to `enabled: false`
**Status:** Draft

## Preconditions

- The environment manifest exists in the Git repository.
- The target component is currently set to `enabled: true` and its resources are running.

## Main Success Scenario

1. Developer opens the environment YAML manifest.
2. Developer sets `enabled: false` for the target application component.
3. Developer commits and pushes the change to the Git repository.
4. ArgoCD detects the commit and triggers reconciliation.
5. Crossplane removes the disabled component from the Environment claim's desired child composite set.
6. If the component had an associated database, Crossplane prunes the database child composite and its managed resources.
7. Kubernetes garbage-collects all associated resources.
8. All component resources are removed from the namespace within 3 minutes.
9. ArgoCD updates environment status to `Synced`.

## Alternative Flows

### A1: Resources Stuck in Terminating State

**Trigger:** One or more resources remain in `Terminating` state beyond 5 minutes (step 7)
**Flow:**

1. System records a stuck-termination event in ArgoCD.
2. ArgoCD status remains `Progressing`.
3. Developer notifies the platform engineer.
4. Platform engineer investigates and force-removes the stuck finalizer.
5. Use case continues at step 8.

### A2: Database Deletion Blocked by Finalizer

**Trigger:** DatabaseInstance has a deletion protection finalizer (step 6)
**Flow:**

1. System records a finalizer-blocked deletion event.
2. Platform engineer reviews data retention policy and removes the finalizer if approved.
3. Use case continues at step 7.

## Postconditions

### Success Postconditions

- No Deployment, Service, ConfigMap, or DatabaseInstance resources exist for the component in the namespace.
- ArgoCD environment status is `Synced`.
- The component remains in the YAML manifest with `enabled: false` for future reference.

### Failure Postconditions

- Residual resources may remain in `Terminating` state.
- ArgoCD status is `Progressing` or `Degraded`.
- Platform engineer is notified for manual intervention.

## Business Rules

### BR-001: Teardown Time SLA

All resources for a disabled component must be fully removed within 3 minutes of ArgoCD detecting the manifest change.

### BR-002: Manifest Preserved

Setting `enabled: false` does not delete the component entry from the YAML. The component definition is preserved so it can be re-enabled later.
