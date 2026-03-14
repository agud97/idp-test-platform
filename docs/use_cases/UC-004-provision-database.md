# Use Case: Provision Database Alongside App

## Overview

**Use Case ID:** UC-004
**Use Case Name:** Provision Database Alongside App
**Primary Actor:** Developer
**Goal:** Automatically provision a managed database instance when an application component is enabled
**Status:** Draft

> **Note:** This use case is triggered automatically as part of UC-002 (Enable Application Component) when the component type has `supports_database: true`. It is not initiated independently.

## Preconditions

- UC-002 is in progress and the component type has `supports_database: true`.
- A supported database engine (PostgreSQL or Redis) is specified in the component definition.
- Crossplane database Composition is registered for the requested engine.

## Main Success Scenario

1. Crossplane reads the `database.engine` and `database.version` fields from the component specification.
2. Crossplane creates a DatabaseInstance custom resource in the environment namespace.
3. System provisions the database pod and persistent storage.
4. Database reaches a healthy running state.
5. System writes the internal cluster DNS hostname to `DatabaseInstance.connection_host`.
6. System creates a Kubernetes Secret containing the database credentials in the environment namespace.
7. Component's Deployment is configured with the database connection details via environment variables sourced from the Secret.
8. System sets DatabaseInstance status to `Ready`.

## Alternative Flows

### A1: Unsupported Database Engine

**Trigger:** Component specifies an engine other than `postgresql` or `redis` (step 1)
**Flow:**

1. System rejects the component with error: "Unsupported database engine: `<engine>`".
2. Developer updates the component definition to use a supported engine or requests a new Composition from the platform engineer.
3. Use case restarts at step 1.

### A2: Persistent Volume Claim Failure

**Trigger:** Kubernetes cannot bind a PersistentVolumeClaim for database storage (step 3)
**Flow:**

1. System records a PVC-binding failure event.
2. DatabaseInstance status is set to `Failed`.
3. Platform engineer checks storage class availability and capacity.
4. Platform engineer resolves the storage issue.
5. Use case restarts at step 3.

### A3: Database Pod Fails to Start

**Trigger:** Database pod crashes or fails readiness probe (step 4)
**Flow:**

1. System records pod failure events.
2. DatabaseInstance status is set to `Failed`.
3. Developer views provisioning logs (UC-014).
4. Developer or platform engineer resolves the configuration issue.
5. Use case restarts at step 3.

## Postconditions

### Success Postconditions

- DatabaseInstance record exists with status `Ready`.
- `connection_host` field is populated with the internal cluster DNS name.
- A Kubernetes Secret with database credentials exists in the environment namespace.
- Application component can connect to the database using the injected credentials.

### Failure Postconditions

- DatabaseInstance record exists with status `Failed`.
- No database pod is running.
- No credentials Secret is created.
- Parent component (UC-002) is also marked as `Degraded`.

## Business Rules

### BR-001: Supported Engines Only

Only PostgreSQL and Redis are supported. Other database engines require a new Crossplane Composition to be registered by a platform engineer before use.

### BR-002: Credentials via Secret

Database credentials must be injected into the application via a Kubernetes Secret. Hardcoded credentials in manifests are not permitted.

### BR-003: No Cloud Provider Dependency

The database must run as an in-cluster workload. External managed database services (AWS RDS, GCP CloudSQL) are not used.
