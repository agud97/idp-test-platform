import express from 'express';
import { promises as fs } from 'fs';
import { coreServices, createBackendPlugin } from '@backstage/backend-plugin-api';
import Router from 'express-promise-router';
import initSqlJs from 'sql.js';

type TeamStatusRow = {
  name: string;
  total: number;
  completed: number;
  validated: number;
  inProgress: number;
  pending: number;
  blocked: number;
  percentage: number;
};

const defaultTrackerPath = '/tmp/idp/tracker.db';
let sqlJsPromise: ReturnType<typeof initSqlJs> | undefined;

function getTrackerPath(): string {
  return process.env.IDP_TRACKER_DB || defaultTrackerPath;
}

async function loadSqlJs() {
  if (!sqlJsPromise) {
    sqlJsPromise = initSqlJs({
      locateFile: file => require.resolve(`sql.js/dist/${file}`),
    });
  }
  return sqlJsPromise;
}

async function readTeamStatus(): Promise<TeamStatusRow[]> {
  const trackerPath = getTrackerPath();
  const sql = await loadSqlJs();
  const bytes = await fs.readFile(trackerPath);
  const db = new sql.Database(bytes);

  try {
    const [result] = db.exec(`
      SELECT
        t.name AS name,
        COUNT(*) AS total,
        SUM(CASE WHEN le.migration_status = 'completed' THEN 1 ELSE 0 END) AS completed,
        SUM(CASE WHEN le.migration_status = 'validated' THEN 1 ELSE 0 END) AS validated,
        SUM(CASE WHEN le.migration_status = 'in_progress' THEN 1 ELSE 0 END) AS in_progress,
        SUM(CASE WHEN le.migration_status = 'pending' THEN 1 ELSE 0 END) AS pending,
        SUM(CASE WHEN le.migration_status = 'blocked' THEN 1 ELSE 0 END) AS blocked
      FROM legacy_environments le
      JOIN teams t ON t.id = le.team_id
      GROUP BY t.name
      ORDER BY t.name
    `);

    if (!result) {
      return [];
    }

    return result.values.map(value => {
      const total = Number(value[1] ?? 0);
      const completed = Number(value[2] ?? 0);
      return {
        name: String(value[0]),
        total,
        completed,
        validated: Number(value[3] ?? 0),
        inProgress: Number(value[4] ?? 0),
        pending: Number(value[5] ?? 0),
        blocked: Number(value[6] ?? 0),
        percentage: total === 0 ? 0 : Math.floor((completed / total) * 100),
      };
    });
  } finally {
    db.close();
  }
}

export const migrationDashboardPlugin = createBackendPlugin({
  pluginId: 'migration',
  register(env) {
    env.registerInit({
      deps: {
        httpRouter: coreServices.httpRouter,
        logger: coreServices.logger,
      },
      async init({ httpRouter, logger }) {
        const router = Router();
        router.use(express.json());

        router.get('/status', async (_req, res) => {
          try {
            const teams = await readTeamStatus();
            res.json({ teams });
          } catch (error) {
            logger.error(`migration status query failed: ${error}`);
            res.status(500).json({
              error: 'migration-status-unavailable',
              message: error instanceof Error ? error.message : String(error),
            });
          }
        });

        httpRouter.use(router);
        httpRouter.addAuthPolicy({
          path: '/status',
          allow: 'unauthenticated',
        });
      },
    });
  },
});
