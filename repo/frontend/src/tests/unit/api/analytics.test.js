import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import { getKPIs, getTrends } from '@/api/analytics.js';

describe('api/analytics.js', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getKPIs sends GET /analytics/kpis with params', async () => { ok({ kpis: [] }); await getKPIs({ range: '30d' }); expect(captured.url).toBe('/analytics/kpis'); expect(captured.params.range).toBe('30d'); });
  it('getTrends sends GET /analytics/trends with params', async () => { ok({ series: [] }); await getTrends({ range: '7d' }); expect(captured.url).toBe('/analytics/trends'); expect(captured.params.range).toBe('7d'); });
  it('getKPIs propagates non-2xx error', async () => {
    client.defaults.adapter = (c) => { const e = new Error(); e.response = { status: 403, data: { code: 'FORBIDDEN', message: 'no scope' }, headers: {}, config: c }; return Promise.reject(e); };
    await expect(getKPIs()).rejects.toMatchObject({ code: 'FORBIDDEN' });
  });
});
