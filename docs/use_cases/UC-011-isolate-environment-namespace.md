# Use Case: Isolate Environment Per Namespace

## Overview

**Use Case ID:** UC-011
**Use Case Name:** Isolate Environment Per Namespace
**Primary Actor:** QA Engineer
**Goal:** Ensure each test environment runs in a dedicated Kubernetes namespace so that environments do not interfere with each other
**Status:** Draft

> **Note:** This use case is executed automatically by the system as part of UC-001 (Submit Environment Manifest). The QA engineer is the primary beneficiary, but no manual action is required beyond submitting the manifest.

## Preconditions

- An environment manifest has been submitted (UC-001 is in progress).
- The environment name is unique across all active environments.
- The cluster's NetworkPolicy controller is operational.

## Main Success Scenario

1. System derives the namespace name from the environment name.
2. System creates a dedicated Kubernetes namespace for the environment.
3. System applies a default NetworkPolicy to the namespace that denies all ingress from other environment namespaces.
4. System applies a NetworkPolicy that allows intra-namespace traffic (pods within the same environment can communicate).
5. System applies a NetworkPolicy that allows egress to cluster-internal DNS.
6. All components of the environment are provisioned exclusively within this namespace.
7. System confirms namespace isolation is active.

## Alternative Flows

### A1: Namespace Name Collision

**Trigger:** A namespace with the derived name already exists in the cluster (step 2)
**Flow:**

1. System rejects the environment creation with error: "Namespace `<name>` already exists".
2. Developer chooses a different environment name.
3. Use case restarts at step 1.

### A2: NetworkPolicy Controller Unavailable

**Trigger:** The cluster's NetworkPolicy controller is not running (step 3)
**Flow:**

1. System creates the namespace but cannot enforce network isolation.
2. System records a warning event: "NetworkPolicy controller unavailable — isolation not enforced".
3. Platform engineer is notified to restore the NetworkPolicy controller.
4. Platform engineer re-applies NetworkPolicies after controller recovery.
5. Use case continues at step 4.

## Postconditions

### Success Postconditions

- A dedicated namespace exists for the environment.
- NetworkPolicies are applied: cross-namespace traffic is blocked, intra-namespace traffic is allowed.
- No pods from other environments can reach this environment's pods via cluster-internal DNS.

### Failure Postconditions

- Namespace may exist without NetworkPolicy enforcement.
- Platform engineer is alerted.
- QA engineer is notified that isolation cannot be guaranteed.

## Business Rules

### BR-001: One Namespace Per Environment

Each environment must have exactly one dedicated namespace. Shared namespaces across environments are not permitted.

### BR-002: Default Deny Policy

The default NetworkPolicy for every environment namespace must deny all ingress from other namespaces. Exceptions must be explicitly declared.

### BR-003: Namespace Naming Convention

Namespace names must follow the pattern `env-<environment-name>` and must comply with Kubernetes DNS label constraints (lowercase, alphanumeric, hyphens, max 63 characters).
