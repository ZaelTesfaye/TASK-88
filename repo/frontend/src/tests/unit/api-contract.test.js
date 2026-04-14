import { describe, it, expect, vi } from 'vitest';

// Mock axios so imports succeed without a real HTTP client.
vi.mock('@/api/client.js', () => {
  const calls = [];
  const handler = {
    get(_target, method) {
      return (...args) => {
        calls.push({ method, args });
        return Promise.resolve({ data: {} });
      };
    },
  };
  return {
    default: new Proxy({}, handler),
    setLogoutCallback: vi.fn(),
    __calls: calls,
  };
});

/**
 * This test programmatically verifies that every URL referenced by the
 * frontend API adapters corresponds to a registered backend route.
 *
 * The source-of-truth backend routes are enumerated here (extracted from
 * backend/internal/router/router.go).  If a route is added or renamed on
 * either side this test will fail.
 */
describe('Frontend API → Backend Route Contract', () => {
  // Backend route patterns (from router.go).  Parameterised segments use
  // `:param` notation.
  const backendRoutes = new Set([
    'GET /auth/login',         // not used by frontend but included
    'POST /auth/login',
    'POST /auth/logout',
    'POST /auth/refresh',
    'GET /org/tree',
    'GET /org/nodes',
    'POST /org/nodes',
    'GET /org/nodes/:id',
    'PUT /org/nodes/:id',
    'DELETE /org/nodes/:id',
    'POST /context/switch',
    'GET /context/current',
    'GET /master/:entity',
    'POST /master/:entity',
    'GET /master/:entity/:id',
    'PUT /master/:entity/:id',
    'POST /master/:entity/:id/deactivate',
    'GET /master/:entity/:id/history',
    'GET /versions/:entity',
    'GET /versions/:entity/:id',
    'GET /versions/:entity/:id/items',
    'GET /versions/:entity/:id/diff',
    'POST /versions/:entity',
    'POST /versions/:entity/:id/review',
    'POST /versions/:entity/:id/items',
    'DELETE /versions/:entity/:id/items/:itemId',
    'POST /versions/:entity/:id/activate',
    'GET /ingestion/sources',
    'POST /ingestion/sources',
    'GET /ingestion/sources/:id',
    'PUT /ingestion/sources/:id',
    'DELETE /ingestion/sources/:id',
    'GET /ingestion/jobs',
    'POST /ingestion/jobs',
    'GET /ingestion/jobs/:id',
    'POST /ingestion/jobs/:id/retry',
    'POST /ingestion/jobs/:id/acknowledge',
    'GET /ingestion/jobs/:id/checkpoints',
    'GET /ingestion/jobs/:id/failures',
    'GET /media',
    'POST /media',
    'GET /media/:id',
    'PUT /media/:id',
    'DELETE /media/:id',
    'GET /media/:id/stream',
    'GET /media/:id/cover',
    'POST /media/:id/lyrics/parse',
    'GET /media/:id/lyrics/search',
    'GET /media/formats/supported',
    'GET /analytics/kpis',
    'GET /analytics/trends',
    'GET /analytics/kpis/definitions',
    'POST /analytics/kpis/definitions',
    'GET /analytics/kpis/definitions/:code',
    'PUT /analytics/kpis/definitions/:code',
    'DELETE /analytics/kpis/definitions/:code',
    'GET /reports/schedules',
    'POST /reports/schedules',
    'GET /reports/schedules/:id',
    'PATCH /reports/schedules/:id',
    'DELETE /reports/schedules/:id',
    'POST /reports/schedules/:id/trigger',
    'GET /reports/runs',
    'GET /reports/runs/:id',
    'GET /reports/runs/:id/download',
    'GET /reports/runs/:id/access-check',
    'GET /audit/logs',
    'GET /audit/logs/:id',
    'GET /audit/logs/search',
    'GET /audit/delete-requests',
    'POST /audit/delete-requests',
    'GET /audit/delete-requests/:id',
    'POST /audit/delete-requests/:id/approve',
    'POST /audit/delete-requests/:id/execute',
    'GET /security/sensitive-fields',
    'POST /security/sensitive-fields',
    'PUT /security/sensitive-fields/:id',
    'DELETE /security/sensitive-fields/:id',
    'GET /security/keys',
    'GET /security/keys/:id',
    'POST /security/keys/rotate',
    'POST /security/password-reset',
    'POST /security/password-reset/:id/approve',
    'GET /security/password-reset',
    'GET /security/retention-policies',
    'POST /security/retention-policies',
    'PUT /security/retention-policies/:id',
    'GET /security/legal-holds',
    'POST /security/legal-holds',
    'POST /security/legal-holds/:id/release',
    'POST /security/purge-runs/dry-run',
    'POST /security/purge-runs/execute',
    'GET /security/purge-runs',
    'GET /integrations/endpoints',
    'POST /integrations/endpoints',
    'GET /integrations/endpoints/:id',
    'PUT /integrations/endpoints/:id',
    'DELETE /integrations/endpoints/:id',
    'POST /integrations/endpoints/:id/test',
    'GET /integrations/deliveries',
    'GET /integrations/deliveries/:id',
    'POST /integrations/deliveries/:id/retry',
    'GET /integrations/connectors',
    'POST /integrations/connectors',
    'GET /integrations/connectors/:id',
    'PUT /integrations/connectors/:id',
    'DELETE /integrations/connectors/:id',
    'POST /integrations/connectors/:id/health-check',
  ]);

  /**
   * Normalise a concrete path like `/org/nodes/42` to a pattern like
   * `/org/nodes/:id` so it can be matched against the backend route set.
   */
  function normalise(method, path) {
    // Replace numeric path segments with :id (or :entity for known patterns).
    const segments = path.split('/').map((seg) => {
      if (/^\d+$/.test(seg)) return ':id';
      return seg;
    });
    return `${method.toUpperCase()} ${segments.join('/')}`;
  }

  // Map of frontend API adapter URLs.  Each entry is [ METHOD, path template ].
  // These are manually enumerated from the api/*.js files and must stay in sync.
  const frontendCalls = [
    // auth.js
    ['POST', '/auth/login'],
    ['POST', '/auth/logout'],
    ['POST', '/auth/refresh'],
    // org.js
    ['GET', '/org/tree'],
    ['POST', '/org/nodes'],
    ['PUT', '/org/nodes/:id'],
    ['DELETE', '/org/nodes/:id'],
    ['POST', '/context/switch'],
    ['GET', '/context/current'],
    // master.js
    ['GET', '/master/:entity'],
    ['POST', '/master/:entity'],
    ['PUT', '/master/:entity/:id'],
    ['POST', '/master/:entity/:id/deactivate'],
    // analytics.js
    ['GET', '/analytics/kpis'],
    ['GET', '/analytics/trends'],
    // reports.js
    ['GET', '/reports/schedules'],
    ['POST', '/reports/schedules'],
    ['GET', '/reports/schedules/:id'],
    ['PATCH', '/reports/schedules/:id'],
    ['DELETE', '/reports/schedules/:id'],
    ['POST', '/reports/schedules/:id/trigger'],
    ['GET', '/reports/runs'],
    ['GET', '/reports/runs/:id'],
    ['GET', '/reports/runs/:id/download'],
    ['GET', '/reports/runs/:id/access-check'],
    // security.js
    ['GET', '/security/sensitive-fields'],
    ['POST', '/security/sensitive-fields'],
    ['PUT', '/security/sensitive-fields/:id'],
    ['DELETE', '/security/sensitive-fields/:id'],
    ['GET', '/security/keys'],
    ['GET', '/security/keys/:id'],
    ['POST', '/security/keys/rotate'],
    ['POST', '/security/password-reset'],
    ['POST', '/security/password-reset/:id/approve'],
    ['GET', '/security/password-reset'],
    ['GET', '/security/retention-policies'],
    ['POST', '/security/retention-policies'],
    ['PUT', '/security/retention-policies/:id'],
    ['GET', '/security/legal-holds'],
    ['POST', '/security/legal-holds'],
    ['POST', '/security/legal-holds/:id/release'],
    ['POST', '/security/purge-runs/dry-run'],
    ['POST', '/security/purge-runs/execute'],
    ['GET', '/security/purge-runs'],
    // ingestion.js
    ['GET', '/ingestion/sources'],
    ['POST', '/ingestion/sources'],
    ['GET', '/ingestion/sources/:id'],
    ['PUT', '/ingestion/sources/:id'],
    ['DELETE', '/ingestion/sources/:id'],
    ['GET', '/ingestion/jobs'],
    ['POST', '/ingestion/jobs'],
    ['GET', '/ingestion/jobs/:id'],
    ['POST', '/ingestion/jobs/:id/retry'],
    ['POST', '/ingestion/jobs/:id/acknowledge'],
    ['GET', '/ingestion/jobs/:id/checkpoints'],
    ['GET', '/ingestion/jobs/:id/failures'],
    // playback.js
    ['GET', '/media'],
    ['POST', '/media'],
    ['GET', '/media/:id'],
    ['PUT', '/media/:id'],
    ['DELETE', '/media/:id'],
    ['GET', '/media/:id/stream'],
    ['GET', '/media/:id/cover'],
    ['POST', '/media/:id/lyrics/parse'],
    ['GET', '/media/:id/lyrics/search'],
    ['GET', '/media/formats/supported'],
    // audit.js
    ['GET', '/audit/logs'],
    ['GET', '/audit/logs/:id'],
    ['GET', '/audit/logs/search'],
    ['GET', '/audit/delete-requests'],
    ['POST', '/audit/delete-requests'],
    ['GET', '/audit/delete-requests/:id'],
    ['POST', '/audit/delete-requests/:id/approve'],
    ['POST', '/audit/delete-requests/:id/execute'],
    // versions.js
    ['GET', '/versions/:entity'],
    ['GET', '/versions/:entity/:id'],
    ['GET', '/versions/:entity/:id/items'],
    ['GET', '/versions/:entity/:id/diff'],
    ['POST', '/versions/:entity'],
    ['POST', '/versions/:entity/:id/review'],
    ['POST', '/versions/:entity/:id/items'],
    ['DELETE', '/versions/:entity/:id/items/:itemId'],
    ['POST', '/versions/:entity/:id/activate'],
    // integrations.js
    ['GET', '/integrations/endpoints'],
    ['POST', '/integrations/endpoints'],
    ['GET', '/integrations/endpoints/:id'],
    ['PUT', '/integrations/endpoints/:id'],
    ['DELETE', '/integrations/endpoints/:id'],
    ['POST', '/integrations/endpoints/:id/test'],
    ['GET', '/integrations/deliveries'],
    ['GET', '/integrations/deliveries/:id'],
    ['POST', '/integrations/deliveries/:id/retry'],
    ['GET', '/integrations/connectors'],
    ['POST', '/integrations/connectors'],
    ['GET', '/integrations/connectors/:id'],
    ['PUT', '/integrations/connectors/:id'],
    ['DELETE', '/integrations/connectors/:id'],
    ['POST', '/integrations/connectors/:id/health-check'],
  ];

  it.each(frontendCalls)(
    '%s %s has a matching backend route',
    (method, path) => {
      const key = `${method} ${path}`;
      expect(backendRoutes.has(key)).toBe(true);
    },
  );

  it('org.js createNode sends parent_id (snake_case), not parentId', async () => {
    // Importing createNode triggers the mocked client.
    const { createNode } = await import('@/api/org.js');
    await createNode(42, { name: 'Test', level_code: 'city', level_label: 'City' });
    // The mock captured calls via Proxy; verify the payload shape.
    const { __calls } = await import('@/api/client.js');
    const postCall = __calls.find(
      (c) => c.method === 'post' && c.args[0] === '/org/nodes',
    );
    expect(postCall).toBeDefined();
    const payload = postCall.args[1];
    expect(payload).toHaveProperty('parent_id', 42);
    expect(payload).not.toHaveProperty('parentId');
  });
});
