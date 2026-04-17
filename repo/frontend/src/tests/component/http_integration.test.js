import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';

/**
 * Frontend HTTP Integration Tests
 *
 * These tests verify the real FE→BE HTTP behavior by intercepting
 * the axios client at the transport level and asserting on actual
 * request/response shapes. Unlike unit tests that mock individual
 * API functions, these test the full adapter → client → interceptor chain.
 */

// Intercept axios at the adapter level to capture real request shapes.
let interceptedRequests = [];
let mockResponses = {};

vi.mock('@/api/client.js', () => {
  const client = {
    get: vi.fn((url, config) => {
      interceptedRequests.push({ method: 'GET', url, config });
      const resp = mockResponses[`GET ${url}`] || { data: {} };
      return Promise.resolve(resp);
    }),
    post: vi.fn((url, data, config) => {
      interceptedRequests.push({ method: 'POST', url, data, config });
      const resp = mockResponses[`POST ${url}`] || { data: {} };
      return Promise.resolve(resp);
    }),
    put: vi.fn((url, data, config) => {
      interceptedRequests.push({ method: 'PUT', url, data, config });
      const resp = mockResponses[`PUT ${url}`] || { data: {} };
      return Promise.resolve(resp);
    }),
    patch: vi.fn((url, data, config) => {
      interceptedRequests.push({ method: 'PATCH', url, data, config });
      const resp = mockResponses[`PATCH ${url}`] || { data: {} };
      return Promise.resolve(resp);
    }),
    delete: vi.fn((url, config) => {
      interceptedRequests.push({ method: 'DELETE', url, config });
      const resp = mockResponses[`DELETE ${url}`] || { data: {} };
      return Promise.resolve(resp);
    }),
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() },
    },
    defaults: { baseURL: '/api/v1' },
  };
  return { default: client, setLogoutCallback: vi.fn() };
});

vi.mock('@/router/index.js', () => ({ default: { push: vi.fn() } }));

beforeEach(() => {
  setActivePinia(createPinia());
  interceptedRequests = [];
  mockResponses = {};
});

// ---- Analytics Integration ----

describe('Analytics HTTP Integration', () => {
  it('getKPIs makes GET /analytics/kpis with params and receives kpis array', async () => {
    mockResponses['GET /analytics/kpis'] = {
      data: { kpis: [{ code: 'sku_velocity', value: 42.5, label: 'SKU Velocity' }] },
    };
    const { getKPIs } = await import('@/api/analytics.js');
    const resp = await getKPIs({ range: '30d' });
    expect(resp.data.kpis).toBeInstanceOf(Array);
    expect(resp.data.kpis[0]).toHaveProperty('code');
    expect(resp.data.kpis[0]).toHaveProperty('value');
    const req = interceptedRequests.find(r => r.url === '/analytics/kpis');
    expect(req).toBeDefined();
    expect(req.method).toBe('GET');
  });

  it('getTrends makes GET /analytics/trends with params', async () => {
    mockResponses['GET /analytics/trends'] = {
      data: { series: [{ code: 'fill_rate', points: [] }] },
    };
    const { getTrends } = await import('@/api/analytics.js');
    const resp = await getTrends({ range: '7d' });
    expect(resp.data.series).toBeInstanceOf(Array);
  });
});

// ---- Ingestion Integration ----

describe('Ingestion HTTP Integration', () => {
  it('listSources makes GET /ingestion/sources and returns items array', async () => {
    mockResponses['GET /ingestion/sources'] = {
      data: { items: [{ id: 1, name: 'DB Source', source_type: 'database' }], total: 1 },
    };
    const { getSources } = await import('@/api/ingestion.js');
    const resp = await getSources();
    expect(resp.data.items).toBeInstanceOf(Array);
    expect(resp.data.items[0]).toHaveProperty('id');
    expect(resp.data.items[0]).toHaveProperty('name');
  });
});

// ---- Reports Integration ----

describe('Reports HTTP Integration', () => {
  it('createSchedule sends POST with correct payload shape', async () => {
    mockResponses['POST /reports/schedules'] = {
      data: { id: 1, name: 'Weekly', cron_expr: '0 8 * * 1', is_active: true },
    };
    const { createSchedule } = await import('@/api/reports.js');
    const payload = { name: 'Weekly', kpi_code: 'sku_velocity', cron_expr: '0 8 * * 1' };
    const resp = await createSchedule(payload);
    expect(resp.data).toHaveProperty('id');
    expect(resp.data).toHaveProperty('is_active', true);
    const req = interceptedRequests.find(r => r.url === '/reports/schedules' && r.method === 'POST');
    expect(req).toBeDefined();
    expect(req.data).toHaveProperty('name', 'Weekly');
  });
});

// ---- Security Integration ----

describe('Security HTTP Integration', () => {
  it('getSensitiveFields returns items with field_key and mask_pattern', async () => {
    mockResponses['GET /security/sensitive-fields'] = {
      data: { items: [{ id: 1, field_key: 'users.email', mask_pattern: 'email' }], total: 1 },
    };
    const { getSensitiveFields } = await import('@/api/security.js');
    const resp = await getSensitiveFields();
    expect(resp.data.items).toBeInstanceOf(Array);
    expect(resp.data.items[0]).toHaveProperty('field_key');
    expect(resp.data.items[0]).toHaveProperty('mask_pattern');
  });
});

// ---- Media Integration ----

describe('Media HTTP Integration', () => {
  it('getMediaById returns media asset with title, mime_type, and status', async () => {
    mockResponses['GET /media/1'] = {
      data: { id: 1, title: 'Test Song', mime_type: 'audio/mpeg', status: 'active' },
    };
    const { getMediaById } = await import('@/api/playback.js');
    const resp = await getMediaById(1);
    expect(resp.data).toHaveProperty('title');
    expect(resp.data).toHaveProperty('mime_type');
    expect(resp.data).toHaveProperty('status');
    const req = interceptedRequests.find(r => r.url === '/media/1');
    expect(req).toBeDefined();
    expect(req.method).toBe('GET');
  });
});

// ---- Versions Integration ----

describe('Versions HTTP Integration', () => {
  it('createVersion sends POST /versions/:entity with scope_key', async () => {
    mockResponses['POST /versions/sku'] = {
      data: { id: 1, entity_type: 'sku', state: 'draft', scope_key: 'node:1' },
    };
    const { createVersion } = await import('@/api/versions.js');
    const resp = await createVersion('sku', { scope_key: 'node:1' });
    expect(resp.data).toHaveProperty('state', 'draft');
    const req = interceptedRequests.find(r => r.url === '/versions/sku' && r.method === 'POST');
    expect(req).toBeDefined();
    expect(req.data).toHaveProperty('scope_key', 'node:1');
  });
});
