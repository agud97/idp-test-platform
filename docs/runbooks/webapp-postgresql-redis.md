# Webapp + PostgreSQL + Redis Migration Runbook

Applies to application type: `webapp-postgresql-redis`
Platform version: `phase-4`

## Outcome

Use this runbook when the legacy application serves HTTP traffic, persists relational data in PostgreSQL, and uses Redis for cache, queue, or session state. The target IDP manifest should contain `webapp`, `postgresql`, and `redis` components.

## Step 1: Export legacy config

1. Run `idp export --compose-file docker-compose.yml --output export/webapp-postgresql-redis`.
2. Confirm the export includes:
   - web container image and exposed port
   - PostgreSQL connection variables
   - Redis host, port, and password settings
3. Record any sidecar or init-container behavior that is not represented directly in compose.

## Step 2: Convert to an IDP manifest

1. Run `idp migrate --compose-file docker-compose.yml --output migrated/environment.yaml`.
2. Review the generated manifest and confirm all three components exist.
3. Ensure the `webapp` component references the internal PostgreSQL and Redis service names produced by the platform.

## Step 3: Review and fix conversion findings

1. Read the conversion report and resolve all flagged items.
2. Check for common fixes:
   - convert custom startup ordering into application retry logic
   - move Redis persistence expectations into application configuration if Redis is used only as cache
   - remove local bind mounts and replace them with image assets or object storage
3. Run `idp validate --file migrated/environment.yaml`.
4. If validation fails, fix the manifest and rerun validation until it passes cleanly.

## Step 4: Submit the manifest and provision the environment

1. Commit the validated manifest into the GitOps repository.
2. Push the change to the watched branch so ArgoCD starts reconciliation.
3. Confirm the namespace, web workload, PostgreSQL composite resources, and Redis resources are created.
4. Wait until the Backstage Kubernetes view shows healthy workloads and ready services.

## Step 5: Validate the migrated environment

1. Test the main user journey through the HTTP endpoint.
2. Confirm PostgreSQL-backed data is readable and writable.
3. Confirm Redis-backed flows behave correctly:
   - cache warmup succeeds
   - session storage works
   - queue consumers drain expected jobs if Redis is used for background work
4. Compare logs and configuration with the legacy environment for any missing dependency wiring.
5. When parity is confirmed, update the tracker record to `validated`.

## Escalation

If Redis behavior differs from legacy and the discrepancy is not explained by this runbook, set the tracker record to `blocked`, attach the failing request or job trace, and escalate to the platform engineer.
