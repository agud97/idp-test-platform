# Use Case: Define Environment Template

## Overview

**Use Case ID:** UC-009
**Use Case Name:** Define Environment Template
**Primary Actor:** Team Lead
**Goal:** Create a reusable environment template in Backstage so that team members can provision consistent, pre-approved application stacks
**Status:** Draft

## Preconditions

- Team lead is authenticated in the Backstage portal with template editor access.
- All component types referenced in the template are registered in the platform catalog.
- The template YAML body has been prepared and validated locally.

## Main Success Scenario

1. Team lead opens the Backstage portal and navigates to the template management section.
2. Team lead selects "Create New Template".
3. System displays the template editor form (name, description, parameterised YAML body).
4. Team lead fills in the template name, description, and parameterised YAML body defining the default component stack.
5. Team lead defines template parameters (e.g., `environment_name`, `image_tag`) that developers will fill in when using the template.
6. Team lead submits the template.
7. System validates the template YAML body and parameter definitions.
8. System saves the template and makes it available in the Backstage scaffolder.
9. System confirms successful publication with a link to the new template.

## Alternative Flows

### A1: Duplicate Template Name

**Trigger:** A template with the same name already exists (step 7)
**Flow:**

1. System displays error: "A template with this name already exists".
2. Team lead chooses a different name or updates the existing template.
3. Use case continues at step 6.

### A2: Template References Unknown Component Type

**Trigger:** The YAML body references a component type not registered in the catalog (step 7)
**Flow:**

1. System displays error: "Unknown component type: `<name>`".
2. Team lead either removes the unknown component from the template or requests its registration (UC-010).
3. Use case continues at step 6.

### A3: Invalid Template Parameter Definition

**Trigger:** A parameter definition is malformed or references a non-existent field in the YAML body (step 7)
**Flow:**

1. System highlights the invalid parameter definition with a descriptive error.
2. Team lead corrects the parameter definition.
3. Use case continues at step 6.

## Postconditions

### Success Postconditions

- Template is saved and visible to all team members in the Backstage scaffolder.
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
