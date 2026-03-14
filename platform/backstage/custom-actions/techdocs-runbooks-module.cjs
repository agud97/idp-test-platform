'use strict';

const fs = require('node:fs/promises');
const path = require('node:path');
const express = require('express');
const Router = require('express-promise-router');
const { marked } = require('marked');
const { createBackendPlugin, coreServices } = require('@backstage/backend-plugin-api');

const runbooksDir = '/app/docs/runbooks';
const runbooks = {
  'webapp-postgresql': {
    title: 'Webapp + PostgreSQL',
    file: 'webapp-postgresql.md',
  },
  'webapp-postgresql-redis': {
    title: 'Webapp + PostgreSQL + Redis',
    file: 'webapp-postgresql-redis.md',
  },
  'worker-redis': {
    title: 'Worker + Redis',
    file: 'worker-redis.md',
  },
};

marked.setOptions({
  headerIds: true,
  mangle: false,
});

function renderLayout({ title, body, activeSlug }) {
  const cards = Object.entries(runbooks).map(([slug, runbook]) => `
    <a class="card${slug === activeSlug ? ' active' : ''}" href="/docs/runbooks/${slug}">
      <span class="eyebrow">Runbook</span>
      <strong>${runbook.title}</strong>
      <span class="slug">${slug}</span>
    </a>
  `).join('');

  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>${title}</title>
    <style>
      :root {
        --bg: #f1efe8;
        --paper: rgba(255, 253, 249, 0.97);
        --ink: #1f2933;
        --muted: #5c6773;
        --line: #d8d2c8;
        --accent: #9a3412;
        --accent-soft: #fff2e8;
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        color: var(--ink);
        background:
          radial-gradient(circle at top right, rgba(154, 52, 18, 0.12), transparent 26rem),
          linear-gradient(180deg, #f8f6f1 0%, var(--bg) 100%);
        font-family: Georgia, "Times New Roman", serif;
      }
      main {
        max-width: 1200px;
        margin: 0 auto;
        padding: 36px 18px 64px;
      }
      header h1 {
        margin: 0;
        font-size: clamp(2rem, 3vw, 3rem);
        line-height: 1.05;
      }
      header p {
        margin: 10px 0 0;
        color: var(--muted);
        font: 500 15px/1.6 "Helvetica Neue", Arial, sans-serif;
      }
      .shell {
        display: grid;
        grid-template-columns: 300px minmax(0, 1fr);
        gap: 22px;
        margin-top: 28px;
      }
      nav {
        align-self: start;
        padding: 18px;
        border: 1px solid var(--line);
        background: var(--paper);
        box-shadow: 0 18px 48px rgba(54, 42, 24, 0.08);
      }
      .nav-title {
        margin: 0 0 14px;
        font: 700 12px/1.4 "Helvetica Neue", Arial, sans-serif;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--muted);
      }
      .card {
        display: block;
        margin-bottom: 12px;
        padding: 14px;
        border: 1px solid var(--line);
        color: inherit;
        text-decoration: none;
        background: #fffdfa;
      }
      .card.active {
        border-color: #d07b5b;
        background: var(--accent-soft);
      }
      .eyebrow, .slug {
        display: block;
        font: 600 12px/1.4 "Helvetica Neue", Arial, sans-serif;
        color: var(--muted);
      }
      .card strong {
        display: block;
        margin: 4px 0;
        font-size: 1.05rem;
      }
      article {
        padding: 28px 32px;
        border: 1px solid var(--line);
        background: var(--paper);
        box-shadow: 0 18px 48px rgba(54, 42, 24, 0.08);
      }
      .notice {
        padding: 18px 20px;
        border: 1px solid #e6b699;
        background: #fff1e8;
        color: #8c2d12;
        font: 700 15px/1.5 "Helvetica Neue", Arial, sans-serif;
      }
      article h1, article h2, article h3 {
        line-height: 1.2;
      }
      article p, article li {
        font-size: 1rem;
      }
      article code {
        padding: 0.12rem 0.32rem;
        background: #f4eee4;
        font-family: "SFMono-Regular", Consolas, monospace;
      }
      article pre {
        overflow-x: auto;
        padding: 14px;
        background: #f4eee4;
      }
      @media (max-width: 860px) {
        .shell {
          grid-template-columns: 1fr;
        }
        article {
          padding: 22px 18px;
        }
      }
    </style>
  </head>
  <body>
    <main>
      <header>
        <h1>TechDocs: Migration Runbooks</h1>
        <p>Self-service migration guidance for the application types currently covered by the platform.</p>
      </header>
      <section class="shell">
        <nav>
          <p class="nav-title">Available Runbooks</p>
          ${cards}
        </nav>
        <article>${body}</article>
      </section>
    </main>
  </body>
</html>`;
}

async function renderRunbookPage(slug) {
  const runbook = runbooks[slug];
  if (!runbook) {
    return renderLayout({
      title: 'Runbook missing',
      body: '<div class="notice">Runbook missing — contact platform engineer</div>',
      activeSlug: null,
    });
  }

  const markdown = await fs.readFile(path.join(runbooksDir, runbook.file), 'utf8');
  return renderLayout({
    title: `${runbook.title} | TechDocs`,
    body: marked.parse(markdown),
    activeSlug: slug,
  });
}

const techdocsRunbooksModule = createBackendPlugin({
  pluginId: 'techdocs-runbooks',
  register(env) {
    env.registerInit({
      deps: {
        rootHttpRouter: coreServices.rootHttpRouter,
      },
      async init({ rootHttpRouter }) {
        const router = Router();
        router.use(express.urlencoded({ extended: false }));

        router.get('/', (_req, res) => {
          res.type('html').send(renderLayout({
            title: 'TechDocs: Migration Runbooks',
            body: `<h2>Choose an application type</h2>
<p>Each runbook follows the UC-018 happy path: export, convert, fix, submit, validate.</p>`,
            activeSlug: null,
          }));
        });

        router.get('/runbooks/:slug', async (req, res) => {
          res.type('html').send(await renderRunbookPage(req.params.slug));
        });

        rootHttpRouter.use('/docs', router);
      },
    });
  },
});

module.exports = techdocsRunbooksModule;
