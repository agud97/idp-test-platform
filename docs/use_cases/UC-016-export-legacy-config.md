# Use Case: Export Legacy Environment Config

## Overview

**Use Case ID:** UC-016
**Use Case Name:** Export Legacy Environment Config
**Primary Actor:** Platform Engineer
**Goal:** Extract the configuration of a legacy docker-compose environment so that its services, environment variables, and volumes are available for migration, even when direct file access is restricted
**Status:** Draft

## Preconditions

- The legacy environment is identified by name or ID.
- Platform engineer has at least read access to the legacy platform API or host.
- The export tool is installed and configured with the necessary credentials.

## Main Success Scenario

1. Platform engineer identifies the legacy environment to export.
2. Platform engineer runs the export tool: `idp export-legacy --env <name> --output legacy-config.yaml`.
3. Tool connects to the legacy platform API or reads available configuration sources (running containers, environment variables, compose files if accessible).
4. Tool extracts: service names, images, exposed ports, environment variables, volume mount paths, and inter-service dependencies.
5. Tool writes the extracted configuration to `legacy-config.yaml`.
6. Tool prints an export summary: number of services extracted, fields that were inaccessible, and recommended manual steps.
7. Platform engineer reviews the export summary and shares `legacy-config.yaml` with the developer for conversion (UC-015).

## Alternative Flows

### A1: Full File Access Denied

**Trigger:** The `docker-compose.yml` file is not accessible but the running containers are (step 3)
**Flow:**

1. Tool switches to container inspection mode: reads config from running container metadata (image, env vars, port bindings, mounts).
2. Tool marks extracted fields as "inferred from runtime" in the export summary.
3. Tool notes that build-time variables or secrets not injected at runtime may be missing.
4. Use case continues at step 5.

### A2: Partial Access — Some Services Inaccessible

**Trigger:** Only a subset of services are accessible due to access restrictions (step 3)
**Flow:**

1. Tool exports only the accessible services.
2. Tool adds inaccessible services to the "Manual Review Required" section of the export summary.
3. Platform engineer documents the gaps and manually reconstructs inaccessible service configs from team knowledge or documentation.
4. Use case continues at step 5.

### A3: No Access at All

**Trigger:** The legacy environment is completely inaccessible (no API, no file, no running containers) (step 3)
**Flow:**

1. Tool exits with error: "Cannot access legacy environment `<name>`. No data extracted."
2. Platform engineer escalates to the legacy platform owner to obtain the configuration.
3. Use case restarts at step 3 after access is granted or configuration is manually provided.

## Postconditions

### Success Postconditions

- A `legacy-config.yaml` file exists with all accessible service configurations.
- An export summary lists accessible, inferred, and missing fields.
- Developer can use the file as input to UC-015 (Convert docker-compose to IDP Manifest).

### Failure Postconditions

- No export file is generated.
- Platform engineer has a clear action plan to obtain the missing configuration manually.

## Business Rules

### BR-001: Non-Destructive Export

The export tool must not modify, stop, or restart any services in the legacy environment.

### BR-002: Sensitive Data Handling

Exported environment variables that appear to be secrets (names containing `PASSWORD`, `SECRET`, `KEY`, `TOKEN`) must be redacted in the output file and flagged for manual re-entry.

### BR-003: Partial Export Is Valid

An incomplete export is a valid and useful output. The tool must not fail silently — every inaccessible field must be explicitly documented in the export summary.
