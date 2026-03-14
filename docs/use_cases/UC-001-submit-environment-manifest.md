# Use Case: Submit Environment Manifest

## Overview

**Use Case ID:** UC-001
**Use Case Name:** Submit Environment Manifest
**Primary Actor:** Developer
**Goal:** Create a new test environment by committing a single YAML manifest to Git
**Status:** Draft

## Preconditions

- Developer has write access to the GitOps repository
- A valid environment YAML manifest has been prepared
- ArgoCD is running and watching the environments path in the repository

## Main Success Scenario

1. Developer prepares an environment YAML manifest listing desired application components with `enabled: true/false` flags.
2. Developer commits the manifest to the designated environments folder in the Git repository.
3. Developer pushes the commit to the remote repository.
4. ArgoCD detects the new commit and triggers a sync.
5. System validates the manifest against the Environment claim/XRD schema.
6. Crossplane reconciles the Environment claim into a dedicated workload namespace, baseline NetworkPolicies, and child component composite resources.
7. System provisions enabled components and their databases from those child composites.
8. ArgoCD reports environment status as `Synced`.
9. Developer receives confirmation that the environment is ready.

## Alternative Flows

### A1: Invalid YAML Schema

**Trigger:** Manifest fails CRD schema validation (step 5)
**Flow:**

1. System rejects the manifest with a human-readable validation error message.
2. ArgoCD sets environment status to `Degraded`.
3. Developer fixes the YAML and pushes a corrected commit.
4. Use case continues at step 4.

### A2: Git Push Rejected

**Trigger:** Developer does not have write permission to the repository branch (step 3)
**Flow:**

1. Git server rejects the push with an authorization error.
2. Developer requests access from the repository owner or platform engineer.
3. Use case restarts at step 3 after access is granted.

### A3: Resource Provisioning Failure

**Trigger:** Crossplane fails to reconcile the Environment claim or one of its child composite resources (step 6)
**Flow:**

1. System records the error in ArgoCD events.
2. ArgoCD sets environment status to `Degraded`.
3. Developer views provisioning logs (UC-014) to diagnose the failure.
4. Developer corrects the manifest or reports the issue to the platform engineer.
5. Use case continues at step 2 after the fix is pushed.

## Postconditions

### Success Postconditions

- Environment record exists in the Backstage software catalog.
- A dedicated Kubernetes namespace is created for the environment.
- All enabled components are running and healthy.
- Git commit SHA is stored in the environment record.

### Failure Postconditions

- No environment resources are created in the cluster.
- ArgoCD status is set to `Degraded` with an error message.
- The Git commit remains in history for audit purposes.

## Business Rules

### BR-001: One Namespace Per Environment

Each environment manifest must result in exactly one dedicated Kubernetes namespace. Sharing namespaces across environments is prohibited.

### BR-002: CRD-Backed Manifest

The environment YAML must conform to the registered Custom Resource Definition. Arbitrary Kubernetes manifests are not accepted.

### BR-003: Git as Single Source of Truth

Direct `kubectl apply` by developers is not permitted. All changes must go through a Git commit.
