# Use Case: Convert docker-compose to IDP Manifest

## Overview

**Use Case ID:** UC-015
**Use Case Name:** Convert docker-compose to IDP Manifest
**Primary Actor:** Developer
**Goal:** Use a CLI tool to automatically convert an existing `docker-compose.yml` into a valid IDP environment YAML manifest to avoid rewriting the configuration from scratch
**Status:** Draft

## Preconditions

- Developer has access to the `docker-compose.yml` file for the legacy environment (obtained directly or via UC-016).
- The IDP migration CLI tool is installed and available.
- The platform component catalog has been reviewed so the developer knows which component types map to which docker-compose services.

## Main Success Scenario

1. Developer runs the migration CLI tool: `idp migrate --from docker-compose.yml --output environment.yaml`.
2. Tool parses the `docker-compose.yml` file and extracts service definitions (image, ports, environment variables, volumes, dependencies).
3. Tool maps each docker-compose service to the closest matching IDP component type from the platform catalog.
4. Tool generates an IDP environment YAML manifest with all services as components, each with `enabled: true`.
5. Tool prints a conversion report listing: successfully mapped services, warnings for unsupported features, and fields requiring manual review.
6. Developer reviews the generated YAML and the conversion report.
7. Developer manually fixes any flagged items and validates the YAML.
8. Developer submits the manifest to Git (UC-001) to provision the migrated environment.

## Alternative Flows

### A1: Service Has No Matching Component Type

**Trigger:** A docker-compose service cannot be mapped to any registered component type (step 3)
**Flow:**

1. Tool marks the service as `enabled: false` with a comment: `# TODO: No matching component type found for service '<name>'`.
2. Tool adds a warning to the conversion report.
3. Developer requests the platform engineer to register a new component type (UC-010) or removes the service if not needed.
4. Use case continues at step 6.

### A2: Unsupported docker-compose Features

**Trigger:** The file uses features with no IDP equivalent (e.g., `build:`, custom networks, `extends:`) (step 2)
**Flow:**

1. Tool skips the unsupported directives and adds them to a "Manual Review Required" section in the conversion report.
2. Developer manually translates the unsupported features or confirms they are not needed.
3. Use case continues at step 6.

### A3: docker-compose File Not Found or Unreadable

**Trigger:** The specified file path does not exist or the developer lacks read permission (step 1)
**Flow:**

1. Tool exits with error: "File not found or not readable: `<path>`".
2. Developer verifies the file path or uses UC-016 to export the configuration first.
3. Use case restarts at step 1.

### A4: Conversion Accuracy Below Threshold

**Trigger:** Fewer than 90% of service definitions are mapped without manual correction (step 5)
**Flow:**

1. Tool marks the overall conversion as "Incomplete" in the report.
2. Developer reviews all warnings and addresses each manually.
3. Developer considers breaking the migration into smaller parts.
4. Use case continues at step 6.

## Postconditions

### Success Postconditions

- A valid IDP environment YAML manifest exists locally, ready for Git submission.
- A conversion report lists what was mapped automatically, what needs manual review, and what was skipped.
- At least 90% of service definitions are correctly mapped without manual correction.

### Failure Postconditions

- No output YAML is generated.
- Tool exits with a clear error message indicating the cause.
- Developer knows the next step to resolve the issue.

## Business Rules

### BR-001: Conversion Accuracy SLA

The CLI tool must correctly map at least 90% of service definitions (image, ports, environment variables, volumes, dependencies) without manual correction.

### BR-002: No Destructive Changes to Legacy Files

The migration tool must never modify the original `docker-compose.yml`. It only reads the source file and writes a new output file.

### BR-003: Partial Results Are Valid Output

If some services cannot be mapped, the tool still outputs a valid (partial) IDP manifest with unmapped services set to `enabled: false` and annotated with TODO comments.
