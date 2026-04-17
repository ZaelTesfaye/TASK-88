import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import { getMasterRecords, createRecord, updateRecord, deactivateRecord } from '@/api/master.js';

describe('api/master.js', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getMasterRecords sends GET /master/:entity with params', async () => { ok({ data: [] }); await getMasterRecords('sku', { page: 2 }); expect(captured.url).toBe('/master/sku'); expect(captured.params.page).toBe(2); });
  it('createRecord sends POST /master/:entity', async () => { ok({ data: { id: 1 } }); await createRecord('sku', { natural_key: 'K1' }); expect(captured.method).toBe('post'); expect(captured.url).toBe('/master/sku'); });
  it('updateRecord sends PUT /master/:entity/:id', async () => { ok(); await updateRecord('sku', 5, { payload_json: '{}' }); expect(captured.url).toBe('/master/sku/5'); expect(captured.method).toBe('put'); });
  it('deactivateRecord sends POST with reason', async () => { ok(); await deactivateRecord('sku', 5, 'obsolete'); expect(captured.url).toBe('/master/sku/5/deactivate'); expect(JSON.parse(captured.data).reason).toBe('obsolete'); });
});
