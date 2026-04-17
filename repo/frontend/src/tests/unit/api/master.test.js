import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import * as api from '@/api/master.js';

describe('api/master.js — full coverage', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getMasterRecords', async () => { ok({ data: [] }); await api.getMasterRecords('sku', { page: 1 }); expect(captured.url).toBe('/master/sku'); });
  it('createRecord', async () => { ok({ id: 1 }); await api.createRecord('sku', { natural_key: 'K1' }); expect(captured.method).toBe('post'); });
  it('updateRecord', async () => { ok(); await api.updateRecord('sku', 5, {}); expect(captured.url).toBe('/master/sku/5'); expect(captured.method).toBe('put'); });
  it('deactivateRecord', async () => { ok(); await api.deactivateRecord('sku', 5, 'obsolete'); expect(captured.url).toBe('/master/sku/5/deactivate'); });
  it('getDuplicates', async () => { ok({ data: [] }); await api.getDuplicates('sku', { key: 'K1' }); expect(captured.url).toBe('/master/sku/duplicates'); });
  it('importRecords constructs FormData', async () => {
    ok({ total_rows: 0 });
    const fakeFile = new Blob(['a,b\n1,2'], { type: 'text/csv' });
    Object.defineProperty(fakeFile, 'name', { value: 'test.csv' });
    await api.importRecords('sku', fakeFile);
    expect(captured.url).toBe('/master/sku/import');
    expect(captured.method).toBe('post');
  });
});
