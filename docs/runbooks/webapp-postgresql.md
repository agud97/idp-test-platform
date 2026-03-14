# Webapp + PostgreSQL Migration Runbook

Applies to application type: `webapp-postgresql`
Platform version: `phase-4`

## Outcome

Use this runbook when the legacy application exposes an HTTP endpoint and stores state in PostgreSQL. The target IDP manifest should contain one `webapp` component and one `postgresql` component.

## Step 1: Export legacy config

1. Run `idp export --compose-file docker-compose.yml --output export/webapp-postgresql`.
2. Confirm the exported bundle contains the application image, exposed port, environment variables, and PostgreSQL connection settings.
3. If the compose file already lives in Git, keep it unchanged and use the export output only as migration evidence.

## Step 2: Convert to an IDP manifest

1. Run `idp migrate --compose-file docker-compose.yml --output migrated/environment.yaml`.
2. Open `migrated/environment.yaml`.
3. Confirm the manifest includes:
   - a `webapp` component with the correct image and port
   - a `postgresql` component with persistence enabled
   - namespace metadata derived from the legacy application name

## Step 3: Review and fix conversion findings

1. Read the conversion report printed by `idp migrate`.
2. Fix any `manual review required` items before submitting:
   - map unsupported compose directives into `spec.components[].config`
   - remove host-bound volumes that are not portable to Kubernetes
   - replace plaintext secrets with references to generated Kubernetes secrets
3. Run `idp validate --file migrated/environment.yaml`.
4. Repeat until validation exits successfully.

## Step 4: Submit the manifest and provision the environment

1. Commit `migrated/environment.yaml` into the GitOps repository under `environments/<team>/<env-name>.yaml`.
2. Add or update the matching catalog entity if the environment should appear in Backstage catalog browsing.
3. Push to the tracked GitOps branch.
4. In Backstage, open the environment page or Kubernetes view and wait for ArgoCD reconciliation to finish.

## Step 5: Validate the migrated environment

1. Verify the web endpoint responds with the same base path and health check behavior as the legacy deployment.
2. Verify PostgreSQL-backed read and write flows succeed with production-like test data.
3. Compare critical settings with the legacy environment:
   - application version
   - replica count
   - database schema availability
   - required environment variables
4. When the migrated environment matches legacy behavior, update the tracker record to `validated`.

## Escalation

If no step in this runbook resolves the issue, set the tracker status to `blocked`, capture the failing command and error text, and contact the platform engineer with the runbook step number.
