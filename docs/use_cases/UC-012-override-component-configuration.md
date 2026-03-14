# Use Case: Override Component Configuration

## Overview

**Use Case ID:** UC-012
**Use Case Name:** Override Component Configuration
**Primary Actor:** Developer
**Goal:** Customize default configuration values for a component (replica count, image tag, environment variables) to test non-default scenarios
**Status:** Draft

## Preconditions

- The environment manifest exists in the Git repository.
- The target component exists in the manifest with `enabled: true`.
- The component type is registered and has defined overrideable parameters.

## Main Success Scenario

1. Developer opens the environment YAML manifest.
2. Developer adds or modifies override fields for the target component: `image_tag`, `replicas`, or `config_overrides` (key-value map).
3. Developer commits and pushes the change to the Git repository.
4. ArgoCD detects the commit and triggers reconciliation.
5. System validates the override values against allowed ranges and known keys.
6. Crossplane applies the overrides to the existing component resources (rolling update for Deployment, config reload for ConfigMap).
7. Component resources are updated without recreating the namespace or other components.
8. ArgoCD updates environment status to `Synced`.

## Alternative Flows

### A1: Replica Count Out of Range

**Trigger:** Developer sets `replicas` to a value outside the allowed range (step 5)
**Flow:**

1. System rejects the manifest with error: "Replica count must be between 0 and 50".
2. Developer corrects the value.
3. Use case continues at step 3.

### A2: Unknown Configuration Key

**Trigger:** A key in `config_overrides` is not recognized by the component type (step 5)
**Flow:**

1. System rejects the manifest with error: "Unknown configuration key: `<key>` for component type `<type>`".
2. Developer checks the component type documentation for valid override keys.
3. Developer corrects the YAML.
4. Use case continues at step 3.

### A3: Invalid Image Tag Format

**Trigger:** `image_tag` does not conform to OCI image tag format (step 5)
**Flow:**

1. System rejects the manifest with error: "Invalid image tag format".
2. Developer corrects the image tag value.
3. Use case continues at step 3.

## Postconditions

### Success Postconditions

- Component Deployment and ConfigMap are updated with the new values.
- Other components in the environment are unaffected.
- ArgoCD environment status is `Synced`.

### Failure Postconditions

- No changes are applied to the cluster.
- ArgoCD status is `Degraded` with a descriptive validation error.

## Business Rules

### BR-001: Scoped Override

Overrides apply only to the specific component. Other components in the same environment remain unchanged.

### BR-002: Replica Range

The `replicas` field must be between 0 and 50. Setting replicas to 0 scales the component down without removing its resources.

### BR-003: Override Precedence

Developer-specified overrides take precedence over the component type's default values defined in the platform catalog.
