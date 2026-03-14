'use strict';

const os = require('node:os');
const path = require('node:path');
const fs = require('fs-extra');
const { createBackendModule, coreServices, resolveSafeChildPath } = require('@backstage/backend-plugin-api');
const { DefaultGithubCredentialsProvider, ScmIntegrations } = require('@backstage/integration');
const { InputError } = require('@backstage/errors');
const {
  createTemplateAction,
  scaffolderActionsExtensionPoint,
  parseRepoUrl,
  cloneRepo,
  commitAndPushRepo,
} = require('@backstage/plugin-scaffolder-node');

function createGithubRepoUpsertAction(options) {
  const { integrations, githubCredentialsProvider } = options;

  return createTemplateAction({
    id: 'github:repo:upsert',
    description: 'Clones an existing GitHub repository branch, copies workspace files into it, commits, and pushes.',
    schema: {
      input: {
        repoUrl: z => z.string({
          description: 'Accepts the format github.com?repo=reponame&owner=owner',
        }),
        branch: z => z.string({
          description: 'The existing target branch to update',
        }),
        sourcePath: z => z.string({
          description: 'Subdirectory of the scaffolder workspace to copy into the repo',
        }).optional(),
        targetPath: z => z.string({
          description: 'Subdirectory within the target repo to write files into',
        }).optional(),
        gitCommitMessage: z => z.string({
          description: 'Commit message for the repository update',
        }).optional(),
        gitAuthorName: z => z.string({
          description: 'Git author name',
        }).optional(),
        gitAuthorEmail: z => z.string({
          description: 'Git author email',
        }).optional(),
        token: z => z.string({
          description: 'Optional GitHub token override',
        }).optional(),
      },
      output: {
        commitHash: z => z.string(),
        remoteUrl: z => z.string(),
        repoContentsUrl: z => z.string(),
      },
    },
    supportsDryRun: true,
    async handler(ctx) {
      const {
        repoUrl,
        branch,
        sourcePath,
        targetPath,
        gitCommitMessage,
        gitAuthorName,
        gitAuthorEmail,
        token: providedToken,
      } = ctx.input;

      const { host, owner, repo } = parseRepoUrl(repoUrl, integrations);
      if (!owner || !repo) {
        throw new InputError(`Invalid repository target ${repoUrl}`);
      }

      const token = providedToken || (await githubCredentialsProvider.getCredentials({
        url: `https://${host}/${encodeURIComponent(owner)}/${encodeURIComponent(repo)}`,
      })).token;

      if (!token) {
        throw new InputError(`No token available for ${owner}/${repo} on ${host}`);
      }

      const remoteUrl = `https://${host}/${owner}/${repo}.git`;
      const repoContentsUrl = `https://${host}/${owner}/${repo}/blob/${branch}`;
      const tempDir = await fs.mkdtemp(path.join(os.tmpdir(), 'scaffolder-upsert-'));

      try {
        if (ctx.isDryRun) {
          ctx.output('commitHash', 'dry-run');
          ctx.output('remoteUrl', remoteUrl);
          ctx.output('repoContentsUrl', repoContentsUrl);
          return;
        }

        await cloneRepo({
          url: remoteUrl,
          dir: tempDir,
          auth: { token },
          logger: ctx.logger,
          ref: branch,
          depth: 1,
        });

        const sourceRoot = sourcePath
          ? resolveSafeChildPath(ctx.workspacePath, sourcePath)
          : ctx.workspacePath;
        const destinationRoot = targetPath
          ? resolveSafeChildPath(tempDir, targetPath)
          : tempDir;

        await fs.ensureDir(destinationRoot);
        await fs.copy(sourceRoot, destinationRoot, {
          overwrite: true,
          errorOnExist: false,
          filter: src => !src.includes(`${path.sep}.git${path.sep}`) && !src.endsWith(`${path.sep}.git`),
        });

        const { commitHash } = await commitAndPushRepo({
          dir: tempDir,
          auth: { token },
          logger: ctx.logger,
          commitMessage: gitCommitMessage || 'Update repository contents',
          gitAuthorInfo: {
            name: gitAuthorName || 'Backstage Scaffolder',
            email: gitAuthorEmail || 'backstage@idp.platform.io',
          },
          branch,
        });

        ctx.output('commitHash', commitHash);
        ctx.output('remoteUrl', remoteUrl);
        ctx.output('repoContentsUrl', repoContentsUrl);
      } finally {
        await fs.remove(tempDir);
      }
    },
  });
}

const gitopsRepoPushModule = createBackendModule({
  pluginId: 'scaffolder',
  moduleId: 'gitops-repo-push',
  register(env) {
    env.registerInit({
      deps: {
        scaffolder: scaffolderActionsExtensionPoint,
        config: coreServices.rootConfig,
      },
      async init({ scaffolder, config }) {
        const integrations = ScmIntegrations.fromConfig(config);
        const githubCredentialsProvider =
          DefaultGithubCredentialsProvider.fromIntegrations(integrations);

        scaffolder.addActions(
          createGithubRepoUpsertAction({
            integrations,
            githubCredentialsProvider,
          }),
        );
      },
    });
  },
});

module.exports = {
  default: gitopsRepoPushModule,
};
