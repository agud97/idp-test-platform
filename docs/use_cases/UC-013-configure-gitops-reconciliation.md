# Use Case: Configure GitOps Reconciliation

## Overview

**Use Case ID:** UC-013
**Use Case Name:** Configure GitOps Reconciliation
**Primary Actor:** Platform Engineer
**Goal:** Set up and maintain ArgoCD to continuously reconcile environment state from Git, automatically correcting any manual drift
**Status:** Draft

## Preconditions

- ArgoCD is installed and running in the cluster.
- The GitOps repository is accessible from the cluster.
- Git credentials (deploy key or token) are configured in ArgoCD.

## Main Success Scenario

1. Platform engineer creates an ArgoCD Application (or ApplicationSet) pointing to the environments folder in the GitOps repository.
2. Platform engineer configures the sync policy: `selfHeal: true`, `automated: true`, `prune: true`.
3. Platform engineer commits the ArgoCD Application manifest to the platform configurations folder in Git.
4. ArgoCD applies the Application manifest and begins watching the environments folder.
5. ArgoCD performs an initial sync, creating any environment resources that do not yet exist in the cluster.
6. ArgoCD enters continuous reconciliation mode: it polls the Git repository every 3 minutes and compares desired state (Git) with live state (cluster).
7. If drift is detected (manual `kubectl apply` or deletion), ArgoCD reverts the cluster to the Git-defined state automatically.
8. Platform engineer verifies reconciliation is working by checking ArgoCD Application health status.

## Alternative Flows

### A1: Invalid Git Credentials

**Trigger:** ArgoCD cannot authenticate to the Git repository (step 1)
**Flow:**

1. ArgoCD Application status shows `Unknown` with an authentication error.
2. Platform engineer updates the Git credentials secret in ArgoCD.
3. Use case continues at step 4.

### A2: Git Repository Unreachable

**Trigger:** Network connectivity to the Git repository is lost (step 6)
**Flow:**

1. ArgoCD pauses reconciliation and records a connectivity error.
2. Last known good state is preserved in the cluster.
3. Platform engineer investigates the network issue.
4. ArgoCD resumes reconciliation automatically once connectivity is restored.
5. Use case continues at step 6.

### A3: Reconciliation Loop Detected

**Trigger:** ArgoCD detects an infinite sync loop (resource is changed back and forth) (step 7)
**Flow:**

1. ArgoCD suspends automated sync for the affected Application.
2. Platform engineer investigates the conflicting resource definition.
3. Platform engineer resolves the conflict in the Git manifest or Crossplane Composition.
4. Platform engineer re-enables automated sync.
5. Use case continues at step 6.

## Postconditions

### Success Postconditions

- ArgoCD continuously watches the GitOps repository for environment changes.
- Any manual drift in the cluster is automatically corrected within 3 minutes.
- All environment Applications show `Synced` and `Healthy` status in ArgoCD.

### Failure Postconditions

- ArgoCD cannot reach Git or the cluster is in an unknown state.
- Platform engineer is alerted via monitoring.
- Manual intervention is required to restore reconciliation.

## Business Rules

### BR-001: Self-Heal Enabled

All environment ArgoCD Applications must have `selfHeal: true` to prevent manual drift from persisting.

### BR-002: Prune Enabled

`prune: true` must be enabled so that resources removed from Git are deleted from the cluster.

### BR-003: ArgoCD Exclusivity

ArgoCD is the sole CD engine for environment management. Other tools must not apply environment manifests directly to the cluster.
