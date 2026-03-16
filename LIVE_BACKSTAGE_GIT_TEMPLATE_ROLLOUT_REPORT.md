# Live Backstage Git-Backed Template Discovery Rollout Report

## Purpose

This report captures the change from image-baked Backstage scaffolder template
discovery to live Git-backed discovery, the related repository/spec updates,
the live rollout sequence, and the final runtime state.

It is intended as an engineering record for future maintenance and for any
operator who needs to understand why the current Backstage template flow works
the way it does.

## Scope

The change covered three areas:

1. Backstage runtime configuration
2. Template/catalog registration model
3. Normative and operational documentation alignment

This report also includes the live rollout work performed against the cluster
and ArgoCD-managed Backstage deployment.

## Initial Problem

Backstage template discovery was previously wired so that the running portal
looked for the `new-environment` template inside the built container image:

- `catalog/all-components.yaml` pointed to `/app/templates/new-environment/template.yaml`
- `platform/backstage/app-config.catalog.yaml` pointed to `/app/catalog/all-components.yaml`
- `platform/backstage/Dockerfile` copied `catalog/` and `templates/` into the image

That model had a practical drawback:

- changing `templates/new-environment/*` in Git did not update the live portal
- a Backstage image rebuild was required for every template change

For a platform that is expected to change frequently and is already GitOps-led,
this was the wrong source-of-truth model.

## Target State

The intended operating model after this change is:

- templates remain authored in the repository under `templates/`
- Backstage discovers templates through catalog URL locations that point to Git
- template changes become visible after Git commit + catalog refresh/reconcile
- Backstage image rebuild is not required for ordinary template changes

The image remains necessary for:

- OIDC JS patching
- custom backend/runtime modules
- TechDocs runbook runtime wiring

But template discovery is no longer image-baked.

## Repository Changes

### 1. Backstage Catalog Bootstrap

Updated:

- `platform/backstage/app-config.catalog.yaml`

Change:

- `catalog.locations` switched from local file target to a Git-backed URL target

Final shape:

```yaml
catalog:
  locations:
    - type: url
      target: https://raw.githubusercontent.com/agud97/idp-test-platform/feature/phase-1-foundation/catalog/all-components.yaml
```

### 2. Template Registration

Updated:

- `catalog/all-components.yaml`

Change:

- `new-environment-template` switched from local file target to Git-backed raw URL

Final shape:

```yaml
kind: Location
spec:
  type: url
  target: https://raw.githubusercontent.com/agud97/idp-test-platform/feature/phase-1-foundation/templates/new-environment/template.yaml
```

### 3. Docker Image Source of Truth Cleanup

Updated:

- `platform/backstage/Dockerfile`

Change:

- removed:
  - `COPY catalog /app/catalog`
  - `COPY templates /app/templates`

Reason:

- the image should no longer imply that `/app/templates` is the runtime source
  of truth for scaffolder discovery

### 4. Backend Reading Allowlist

Updated:

- `platform/backstage/helm-values.yaml`

Change:

- added:

```yaml
backend:
  reading:
    allow:
      - host: raw.githubusercontent.com
```

Reason:

- Backstage backend blocks remote reads unless the host is explicitly allowed

## Documentation / Spec Alignment

Updated documents:

- `docs/requirements.md`
- `docs/constraints.md`
- `docs/acceptance_criteria.md`
- `docs/use_cases/UC-005-create-environment-via-portal.md`
- `docs/use_cases/UC-009-define-environment-template.md`
- `docs/plan.yaml`
- `docs/POST_IMPLEMENTATION_DOC_CHANGES.md`
- `docs/ARCHITECTURE_OVERVIEW.md`
- `docs/NEXT_AGENT_RISKS_AND_RESOLVED_BLOCKERS.md`
- `docs/status.md`
- `SESSION_BOOTSTRAP.md`
- `docs/BACKSTAGE_OPERATIONS_GUIDE.md`

Key semantic updates:

- template discovery is now defined as Git-backed
- image-baked `/app/templates` is explicitly not the source of truth
- UC-009 now models template definition as a Git change, not a portal-side save
- AC-045 now expects catalog refresh/discovery without image rebuild
- task-4.3 wording in `plan.yaml` now matches live Git-backed discovery

## Additional Template Work Included In The Same Session

The session also added `account-api` support to the `new-environment` template.

Files changed:

- `templates/new-environment/template.yaml`
- `templates/new-environment/skeleton/environment.yaml`

Capabilities added:

- `account_api_enabled`
- `account_api_image_tag`
- `account_api_replicas`
- typed fields for multiple safe `configOverrides`

This allows Backstage to render per-environment `account-api` settings through
the form-driven GitOps flow.

## Live Rollout Timeline

The rollout was performed against the ArgoCD-managed `backstage` Application in
namespace `argocd`, targeting runtime namespace `backstage`.

### Step 1. Confirm ArgoCD wiring

Verified:

- `platform/argocd/app-backstage.yaml` already used repo-hosted valueFiles
- Backstage was sourcing `helm-values.yaml` and `app-config.catalog.yaml` from
  the Git repository branch `feature/phase-1-foundation`

Implication:

- no manual chart patching was required
- only Git commit + ArgoCD refresh/sync was needed

### Step 2. First live sync to Git-backed URL config

Observed revision:

- `7cd753a898f7b97f2bf31ef6e311bfabe26cf303`

Result:

- ArgoCD synced successfully
- `backstage-app-config` updated from:
  - `type: file`
  - `target: /app/catalog/all-components.yaml`
- to:
  - `type: url`
  - `target: https://github.com/.../blob/...`

Problem discovered:

- Backstage backend could not read `github.com/.../blob/...`

Observed runtime log:

- `Unable to read url, no matching files found`

### Step 3. Switch from `blob` to raw content URLs

Fix committed:

- use `https://raw.githubusercontent.com/...`

Observed revision:

- `0c29520138a2b91716a44b1a6e379447b6d0e666`

Result:

- ArgoCD synced successfully
- live config updated to raw URLs

Problem discovered:

- Backstage backend still rejected the URL host

Observed runtime log:

- `NotAllowedError: Reading from 'https://raw.githubusercontent.com/...' is not allowed`

### Step 4. Allow raw GitHub reads

Fix committed:

- `backend.reading.allow: raw.githubusercontent.com`

Observed revision:

- `2c57e58c82764d695102d090c3d4787aac64e37d`

Result:

- ArgoCD synced successfully
- live `backstage-app-config` included:
  - raw GitHub catalog target
  - `backend.reading.allow`

### Step 5. Deployment rollouts

Performed:

- repeated `kubectl rollout restart deployment/backstage -n backstage`
- waited for rollout completion after each config change

Final running pod observed:

- `backstage-8598bbb7c8-6tf9c`

## Live Validation Results

### ArgoCD

Validated:

- `Application/backstage` remained `Synced` and `Healthy`
- latest synced value-source revision reached:
  - `2c57e58c82764d695102d090c3d4787aac64e37d`

### Runtime Config

Validated in `ConfigMap/backstage-app-config`:

- `catalog.locations[0].type = url`
- `catalog.locations[0].target = raw.githubusercontent.com/.../catalog/all-components.yaml`
- `backend.reading.allow` includes `raw.githubusercontent.com`

### Backstage Logs

Final validation result:

- the previous errors disappeared:
  - no `Unable to read url`
  - no `NotAllowedError`
- catalog/scaffolder initialization completed
- software catalog collation completed successfully

This is the key runtime proof that Backstage can now read the Git-backed
catalog location.

## Remaining Practical Limitations

The following points still apply:

1. Backstage API from outside the portal requires authentication.
   - Direct unauthenticated `curl` to catalog endpoints returned `401`.
   - Because of that, final UI/entity confirmation through external API was not
     completed in this session.

2. Template changes no longer require image rebuild, but they still require:
   - a Git commit to the tracked branch
   - Backstage catalog refresh / ArgoCD reconcile of config when relevant

3. The current `account-api` form support is typed and explicit.
   - It supports the fields modeled in the template.
   - It is not a generic arbitrary key-value `configOverrides` editor.

4. `imageRepository` for `account-api` is still not exposed in the form.
   - `imageTag` is exposed
   - repository-level image override still needs Git editing if required

## Commits Produced During This Work

### Template + CLI + account-api changes

- `ba2e82f` — `Support account-api scaffolder overrides and safer migrate output`
- `b451752` — `Document account-api component onboarding flow`

### Git-backed template discovery changes

- `7cd753a` — `Switch Backstage template discovery to live Git catalog`
- `0c29520` — `Use raw GitHub URLs for Backstage catalog targets`
- `2c57e58` — `Allow Backstage to read raw GitHub catalog URLs`

## Final Conclusion

The platform is now operating with live Git-backed Backstage template
discovery.

What is true after this change:

- the running portal no longer relies on baked `/app/templates` for template discovery
- Backstage reads the bootstrap catalog from Git via raw GitHub content URLs
- template discovery no longer requires a Backstage image rebuild
- the `new-environment` template remains GitOps-driven
- `account-api` can be modeled through the Backstage template with
  per-environment override fields

Operationally, the main architectural correction is complete:

- Backstage image = runtime shell and custom wiring
- Git repository = source of truth for scaffolder templates and catalog bootstrap

That split is the correct one for a frequently changing GitOps-managed platform.
