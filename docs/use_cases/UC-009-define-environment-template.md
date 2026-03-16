# Use Case: Define Environment Template

## Overview

**Use Case ID:** UC-009
**Use Case Name:** Define Environment Template
**Primary Actor:** Team Lead
**Goal:** Create a reusable environment template in Git so that Backstage can discover it live and team members can provision consistent, pre-approved application stacks
**Status:** Draft

## Preconditions

- Team lead has write access to the GitOps repository path that stores Backstage templates.
- All component types referenced in the template are registered in the platform catalog.
- The template YAML body has been prepared and validated locally.

## Main Success Scenario

1. Team lead creates or updates a template under `templates/<template-name>/` in the GitOps repository.
2. Team lead updates the Backstage catalog registration so the template is reachable through a Git-backed location.
3. Team lead commits and pushes the template change to the GitOps branch.
4. Backstage refreshes its catalog locations.
5. System validates the template entity and parameter schema, including any conditional service-selection fields.
6. System makes the template available in the Backstage scaffolder without a portal image rebuild.
7. Team lead confirms successful publication by opening the new template in Backstage.

## Alternative Flows

### A1: Duplicate Template Name

**Trigger:** A template with the same name already exists (step 5)
**Flow:**

1. System displays error: "A template with this name already exists".
2. Team lead chooses a different name or updates the existing template in Git.
3. Use case continues at step 3.

### A2: Template References Unknown Component Type

**Trigger:** The YAML body references a component type not registered in the catalog (step 5)
**Flow:**

1. System displays error: "Unknown component type: `<name>`".
2. Team lead either removes the unknown component from the template or requests its registration (UC-010).
3. Use case continues at step 3.

### A3: Invalid Template Parameter Definition

**Trigger:** A parameter definition is malformed or references a non-existent field in the YAML body (step 5)
**Flow:**

1. System highlights the invalid parameter definition with a descriptive error.
2. Team lead corrects the parameter definition in Git.
3. Use case continues at step 3.

## Postconditions

### Success Postconditions

- Template is committed to Git and visible to all team members in the Backstage scaffolder.
- Developers can use the template to create environments via UC-005.

### Failure Postconditions

- Template is not saved.
- System displays a specific error message guiding the team lead to fix the issue.

## Business Rules

### BR-001: Approved Components Only

Templates may only reference component types that are registered in the platform catalog. Ad-hoc component definitions are not permitted.

### BR-002: Template Ownership

Each template has a designated owner (team lead) who is responsible for maintaining and updating it.

### BR-003: Parameterisation Required

Any value that developers are expected to customise (environment name, image tag) must be exposed as a named template parameter. Hard-coded developer-specific values are not permitted.

### BR-004: Git-Backed Discoverability

The template must be reachable through a Git/URL-backed Backstage catalog location. A template that exists only inside a built portal image is not sufficient.

### BR-005: Scalable Service Selection

Templates intended for multi-service environments should model service enablement as a unified selection step with conditional fields for enabled services, so the form remains usable as the number of supported services grows.
