# Use Case: Register New Component Type

## Overview

**Use Case ID:** UC-010
**Use Case Name:** Register New Component Type
**Primary Actor:** Platform Engineer
**Goal:** Add a new application component type to the platform catalog so that developers can include it in environment manifests without schema changes
**Status:** Draft

## Preconditions

- Platform engineer has write access to the GitOps repository and the Crossplane composition registry.
- A Crossplane Composition defining the new component's Kubernetes resources has been written and tested.
- The component type name follows the platform naming convention.

## Main Success Scenario

1. Platform engineer writes and tests the Crossplane Composition YAML for the new component type.
2. Platform engineer commits the Composition to the platform configurations folder in the GitOps repository.
3. ArgoCD deploys the new Composition to the cluster.
4. Platform engineer adds the component type definition (name, description, default image, default replicas, supports_database flag) to the platform catalog configuration.
5. System validates the catalog entry and registers the new component type.
6. Platform engineer verifies the component type is available by submitting a test environment manifest referencing it.
7. New component type is now available for developers to use in their environment YAML manifests and Backstage templates.

## Alternative Flows

### A1: Composition Validation Failure

**Trigger:** The Crossplane Composition YAML is invalid or references undefined APIs (step 3)
**Flow:**

1. ArgoCD sets the Composition sync status to `Degraded`.
2. Platform engineer views the ArgoCD error and fixes the Composition YAML.
3. Platform engineer pushes the corrected Composition.
4. Use case continues at step 3.

### A2: Duplicate Component Type Name

**Trigger:** A component type with the same name already exists in the catalog (step 5)
**Flow:**

1. System rejects the entry with error: "Component type `<name>` is already registered".
2. Platform engineer either updates the existing type or chooses a unique name.
3. Use case continues at step 4.

### A3: Verification Test Fails

**Trigger:** Test environment using the new component type does not reach `Synced` status (step 6)
**Flow:**

1. Platform engineer inspects provisioning logs.
2. Platform engineer identifies and fixes the issue in the Composition or catalog entry.
3. Use case continues at step 2.

## Postconditions

### Success Postconditions

- Crossplane Composition is deployed and active in the cluster.
- Component type is listed in the platform catalog.
- Developers can reference the new component type in environment manifests immediately.
- Existing templates can be updated to include the new component type.

### Failure Postconditions

- Composition is not deployed or is in an error state.
- Component type is not added to the catalog.
- Developers cannot use the new type until the issue is resolved.

## Business Rules

### BR-001: Composition-Backed Types Only

Every component type must have a corresponding Crossplane Composition. Component types without a Composition cannot be registered.

### BR-002: No Schema Breaking Changes

Adding a new component type must not change the Environment claim/XRD schema in a way that invalidates existing environment manifests.

### BR-003: Platform Engineer Authority

Only platform engineers may register new component types. Developers and team leads may request new types but cannot register them directly.
