# Worker + Redis Migration Runbook

Applies to application type: `worker-redis`
Platform version: `phase-4`

## Outcome

Use this runbook when the legacy application runs background jobs and depends on Redis as its only managed service. The target IDP manifest should contain a `webapp` component configured as a worker or a dedicated worker component pattern accepted by the current platform, plus a `redis` component.

## Step 1: Export legacy config

1. Run `idp export --compose-file docker-compose.yml --output export/worker-redis`.
2. Confirm the export captures:
   - worker image and command
   - Redis connection settings
   - concurrency, queue name, and retry settings
3. Save the export output with the migration ticket so the original runtime parameters are preserved.

## Step 2: Convert to an IDP manifest

1. Run `idp migrate --compose-file docker-compose.yml --output migrated/environment.yaml`.
2. Open the generated manifest and confirm:
   - the worker workload command or args were preserved
   - Redis component exists and is enabled
   - no HTTP ingress is enabled unless the worker also exposes an admin endpoint

## Step 3: Review and fix conversion findings

1. Resolve any unsupported compose features reported by the converter.
2. Pay special attention to:
   - restart policies
   - queue concurrency flags
   - environment-specific cron or scheduler settings
3. Run `idp validate --file migrated/environment.yaml`.
4. Keep iterating until the manifest validates without errors.

## Step 4: Submit the manifest and provision the environment

1. Commit the validated manifest into the GitOps repository and push it to the tracked branch.
2. Wait for ArgoCD to create the namespace and worker workload.
3. Confirm the workload starts successfully and the Redis service is reachable from the pod.
4. If the migration is tracked in Backstage, verify the environment entity appears and the Kubernetes tab shows the live worker resources.

## Step 5: Validate the migrated environment

1. Enqueue representative jobs and confirm they are consumed successfully.
2. Compare queue latency, retry behavior, and error handling with the legacy worker.
3. Verify Redis keys or streams are created as expected.
4. Review pod logs for startup errors, connection retries, or dropped jobs.
5. When the migrated worker processes jobs correctly, update the tracker record to `validated`.

## Escalation

If the worker cannot process jobs after the manifest validates, set the tracker status to `blocked`, capture the failing job payload or log excerpt, and contact the platform engineer.
