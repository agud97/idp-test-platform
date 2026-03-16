# Use Case: Create Environment via Portal

## Overview

**Use Case ID:** UC-005
**Use Case Name:** Create Environment via Portal
**Primary Actor:** Developer
**Goal:** Provision a new test environment through the Backstage UI without writing YAML manually, using a template discovered live from the GitOps repository
**Status:** Draft

> **Note:** This use case includes UC-001 (Submit Environment Manifest) — the portal generates and commits the manifest on the developer's behalf.

## Preconditions

- Developer is authenticated in the Backstage portal.
- At least one environment template is available in the Backstage scaffolder via the Git-backed Backstage catalog.
- Developer has write access to the GitOps repository (Backstage acts on their behalf via a service token).

## Main Success Scenario

1. Developer opens the Backstage portal and navigates to "Create" → "New Test Environment".
2. System displays available environment templates discovered from the Git-backed Backstage catalog.
3. Developer selects an appropriate template.
4. System displays a form with configurable parameters (environment name, component selections, configuration overrides).
5. Developer fills in the form and submits.
6. System validates the form input.
7. System generates a valid environment YAML manifest from the template and form values.
8. System commits the manifest to the GitOps repository in the designated environments folder.
9. System registers the new environment in the Backstage software catalog.
10. UC-001 (Submit Environment Manifest) executes: ArgoCD detects the commit and provisions the environment.
11. System displays a success message with a link to the environment catalog entry.

## Alternative Flows

### A1: No Templates Available

**Trigger:** No environment templates exist in the Backstage scaffolder (step 2)
**Flow:**

1. System displays "No templates available" message.
2. Developer requests a template from the team lead (UC-009).
3. Use case restarts at step 2 after a template is created.

### A2: Form Validation Error

**Trigger:** Required form fields are empty or contain invalid values (step 6)
**Flow:**

1. System highlights invalid fields with descriptive error messages.
2. Developer corrects the input.
3. Use case continues at step 6.

### A3: Environment Name Already Taken

**Trigger:** An environment with the same name already exists for this owner (step 6)
**Flow:**

1. System displays error: "Environment name already exists. Choose a different name."
2. Developer changes the environment name.
3. Use case continues at step 6.

### A4: Git Commit Fails

**Trigger:** Backstage cannot commit to the GitOps repository (step 8)
**Flow:**

1. System displays an error message indicating the Git operation failed.
2. Platform engineer verifies Backstage Git credentials and repository access.
3. Use case restarts at step 7 after the issue is resolved.

## Postconditions

### Success Postconditions

- Environment manifest is committed to the GitOps repository.
- Environment entry is registered in the Backstage software catalog with owner and team metadata.
- ArgoCD begins reconciling the environment (status transitions to `Progressing`).

### Failure Postconditions

- No manifest is committed to Git.
- No environment entry is added to the catalog.
- System displays an actionable error message to the developer.

## Business Rules

### BR-001: Template-Driven Generation

The portal must generate YAML only from approved templates. Arbitrary YAML input from the form is not accepted.

### BR-002: Ownership Registration

Every environment created via the portal must have an `owner` (developer) and a `team` recorded in the catalog entry.

### BR-003: Name Uniqueness

Environment names must be unique per owner to prevent namespace collisions.
