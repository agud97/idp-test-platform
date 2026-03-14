'use strict';

const express = require('express');
const fs = require('node:fs/promises');
const Router = require('express-promise-router');
const initSqlJs = require('sql.js');
const { createBackendPlugin, coreServices } = require('@backstage/backend-plugin-api');

const defaultTrackerPath = '/tmp/idp/tracker.db';
let sqlJsPromise;

function getTrackerPath() {
  return process.env.IDP_TRACKER_DB || defaultTrackerPath;
}

function loadSqlJs() {
  if (!sqlJsPromise) {
    sqlJsPromise = initSqlJs({
      locateFile: file => require.resolve(`sql.js/dist/${file}`),
    });
  }
  return sqlJsPromise;
}

async function readTeamStatus() {
  const sql = await loadSqlJs();
  const bytes = await fs.readFile(getTrackerPath());
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

const migrationDashboardModule = createBackendPlugin({
  pluginId: 'migration',
  register(env) {
    env.registerInit({
      deps: {
        httpRouter: coreServices.httpRouter,
        rootHttpRouter: coreServices.rootHttpRouter,
        logger: coreServices.logger,
      },
      async init({ httpRouter, rootHttpRouter, logger }) {
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

        rootHttpRouter.use('/migration', (_req, res) => {
          res.type('html').send(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Migration Dashboard</title>
    <style>
      :root {
        color-scheme: light;
        --bg: #f4efe7;
        --panel: #fffdf8;
        --ink: #1f2933;
        --muted: #58636f;
        --line: #d8cfc2;
        --accent: #0f766e;
        --accent-soft: #dff4ef;
        --warn: #92400e;
        --warn-soft: #fef3c7;
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        font-family: Georgia, "Times New Roman", serif;
        color: var(--ink);
        background:
          radial-gradient(circle at top left, #efe2cf 0, transparent 28rem),
          linear-gradient(180deg, #f7f3ec 0%, var(--bg) 100%);
      }
      main {
        max-width: 1100px;
        margin: 0 auto;
        padding: 48px 20px 64px;
      }
      h1 {
        margin: 0 0 8px;
        font-size: clamp(2rem, 3vw, 3.25rem);
        line-height: 1;
      }
      p {
        margin: 0;
        color: var(--muted);
        font-family: "Helvetica Neue", Arial, sans-serif;
      }
      .banner {
        display: none;
        margin-top: 24px;
        padding: 14px 16px;
        border: 1px solid #6aa89c;
        background: var(--accent-soft);
        color: #0b4f4a;
        font: 700 15px/1.4 "Helvetica Neue", Arial, sans-serif;
        letter-spacing: 0.02em;
        text-transform: uppercase;
      }
      .shell {
        margin-top: 28px;
        border: 1px solid var(--line);
        background: rgba(255, 253, 248, 0.92);
        box-shadow: 0 18px 60px rgba(78, 57, 33, 0.12);
        overflow: hidden;
      }
      .statusbar {
        display: flex;
        justify-content: space-between;
        gap: 16px;
        padding: 14px 18px;
        border-bottom: 1px solid var(--line);
        font: 600 13px/1.4 "Helvetica Neue", Arial, sans-serif;
        color: var(--muted);
      }
      .error {
        display: none;
        margin-top: 16px;
        padding: 14px 16px;
        border: 1px solid #e5b38b;
        background: var(--warn-soft);
        color: var(--warn);
        font: 600 14px/1.4 "Helvetica Neue", Arial, sans-serif;
      }
      table {
        width: 100%;
        border-collapse: collapse;
      }
      thead th {
        padding: 14px 18px;
        text-align: left;
        border-bottom: 1px solid var(--line);
        font: 700 12px/1.4 "Helvetica Neue", Arial, sans-serif;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--muted);
      }
      tbody td {
        padding: 16px 18px;
        border-bottom: 1px solid #ece3d7;
        font: 500 15px/1.4 "Helvetica Neue", Arial, sans-serif;
      }
      tbody tr:last-child td { border-bottom: 0; }
      .empty {
        padding: 28px 18px;
        color: var(--muted);
        font: 500 15px/1.5 "Helvetica Neue", Arial, sans-serif;
      }
      .percent {
        font-weight: 700;
        color: var(--accent);
      }
      @media (max-width: 720px) {
        main { padding: 28px 12px 40px; }
        .shell { overflow-x: auto; }
        table { min-width: 760px; }
      }
    </style>
  </head>
  <body>
    <main>
      <h1>Migration Dashboard</h1>
      <p>Live team progress derived from legacy environment tracker records.</p>
      <div id="banner" class="banner">Gate PASSED</div>
      <div id="error" class="error"></div>
      <section class="shell">
        <div class="statusbar">
          <span>Route: /migration</span>
          <span id="updated">Refreshing…</span>
        </div>
        <table>
          <thead>
            <tr>
              <th>Team</th>
              <th>Total</th>
              <th>Completed</th>
              <th>Validated</th>
              <th>In Progress</th>
              <th>Blocked</th>
              <th>%</th>
            </tr>
          </thead>
          <tbody id="rows">
            <tr><td colspan="7" class="empty">Loading migration status…</td></tr>
          </tbody>
        </table>
      </section>
    </main>
    <script>
      const rows = document.getElementById('rows');
      const banner = document.getElementById('banner');
      const error = document.getElementById('error');
      const updated = document.getElementById('updated');

      function renderRows(teams) {
        if (!teams.length) {
          rows.innerHTML = '<tr><td colspan="7" class="empty">No legacy environments registered.</td></tr>';
          banner.style.display = 'none';
          return;
        }
        rows.innerHTML = teams.map(team => \`
          <tr>
            <td>\${team.name}</td>
            <td>\${team.total}</td>
            <td>\${team.completed}</td>
            <td>\${team.validated}</td>
            <td>\${team.inProgress}</td>
            <td>\${team.blocked}</td>
            <td class="percent">\${team.percentage}%</td>
          </tr>
        \`).join('');
        const gatePassed = teams.every(team => team.total > 0 && team.completed === team.total);
        banner.style.display = gatePassed ? 'block' : 'none';
      }

      async function refresh() {
        try {
          const response = await fetch('/api/migration/status', { cache: 'no-store' });
          if (!response.ok) {
            throw new Error('status request failed: ' + response.status);
          }
          const payload = await response.json();
          error.style.display = 'none';
          renderRows(payload.teams || []);
          updated.textContent = 'Updated ' + new Date().toLocaleTimeString();
        } catch (err) {
          error.textContent = err.message || String(err);
          error.style.display = 'block';
          updated.textContent = 'Refresh failed';
        }
      }

      refresh();
      window.setInterval(refresh, 60000);
    </script>
  </body>
</html>`);
        });
      },
    });
  },
});

module.exports = {
  default: migrationDashboardModule,
};
