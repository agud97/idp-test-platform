# Use Case: Delete Environment

## Overview

**Use Case ID:** UC-008
**Use Case Name:** Delete Environment
**Primary Actor:** Developer
**Goal:** Permanently remove a test environment and all its Kubernetes resources by removing its manifest from Git
**Status:** Draft

## Preconditions

- Developer is the owner of the environment or is a platform engineer.
- The environment manifest exists in the GitOps repository.
- Developer has write access to the GitOps repository.

## Main Success Scenario

1. Developer locates the environment manifest in the GitOps repository.
2. Developer deletes or archives the manifest file from the environments folder.
3. Developer commits and pushes the change.
4. ArgoCD detects the removal and triggers deletion of the ArgoCD Application.
5. Crossplane deletes all Kubernetes resources associated with the environment (Deployments, Services, ConfigMaps, DatabaseInstances, Secrets).
6. Kubernetes garbage-collects all resources in the environment namespace.
7. Namespace is deleted.
8. System removes the environment entry from the Backstage software catalog.
9. ArgoCD Application record is removed.

## Alternative Flows

### A1: Unauthorized Deletion Attempt

**Trigger:** A developer who does not own the environment attempts to delete it (step 2)
**Flow:**

1. Git repository rejects the push due to branch protection rules.
2. Developer is informed they do not have permission to modify this environment.
3. Developer contacts the environment owner or platform engineer.
4. Use case ends without deletion.

### A2: Resources Stuck in Terminating State

**Trigger:** One or more resources remain in `Terminating` state beyond 5 minutes (step 6)
**Flow:**

1. Platform engineer is notified of the stuck resources.
2. Platform engineer investigates and removes stuck finalizers manually.
3. Use case continues at step 7.

### A3: Namespace Deletion Blocked

**Trigger:** Namespace has a deletion protection annotation or finalizer (step 7)
**Flow:**

1. System records the blocked deletion event.
2. Platform engineer removes the protection annotation.
3. Use case continues at step 7.

## Postconditions

### Success Postconditions

- All Kubernetes resources (pods, services, configmaps, PVCs, secrets) for the environment are deleted.
- Environment namespace no longer exists in the cluster.
- Environment entry is removed from the Backstage catalog.
- Manifest commit remains in Git history for audit purposes.

### Failure Postconditions

- Some resources may remain in `Terminating` state.
- Namespace may still exist with orphaned resources.
- Platform engineer must intervene to complete cleanup.

## Business Rules

### BR-001: Owner-Only Deletion

Only the environment owner or a platform engineer may delete an environment. This is enforced at the Git repository level via branch protection.

### BR-002: Audit Trail Preserved

The deleted manifest commit must remain in Git history. Permanent erasure from Git history is not permitted.

### BR-003: No Partial Deletion

Deletion is all-or-nothing. Selectively removing individual resources from a running environment must be done via UC-003 (Disable Application Component), not by deleting resource files.
