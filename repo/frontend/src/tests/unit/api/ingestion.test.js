import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import { getSources, createSource, getSource, updateSource, deleteSource } from '@/api/ingestion.js';

describe('api/ingestion.js', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getSources sends GET /ingestion/sources', async () => { ok({ items: [] }); await getSources(); expect(captured.url).toBe('/ingestion/sources'); });
  it('createSource sends POST /ingestion/sources', async () => { ok({ id: 1 }); await createSource({ name: 'S', source_type: 'database' }); expect(captured.method).toBe('post'); expect(captured.url).toBe('/ingestion/sources'); });
  it('getSource sends GET /ingestion/sources/:id', async () => { ok({ id: 3 }); await getSource(3); expect(captured.url).toBe('/ingestion/sources/3'); });
  it('updateSource sends PUT /ingestion/sources/:id', async () => { ok(); await updateSource(3, { name: 'X' }); expect(captured.url).toBe('/ingestion/sources/3'); expect(captured.method).toBe('put'); });
  it('deleteSource sends DELETE /ingestion/sources/:id', async () => { ok(); await deleteSource(3); expect(captured.url).toBe('/ingestion/sources/3'); expect(captured.method).toBe('delete'); });
  it('createSource propagates 400 error', async () => {
    client.defaults.adapter = (c) => { const e = new Error(); e.response = { status: 400, data: { code: 'BAD_REQUEST', message: 'missing name' }, headers: {}, config: c }; return Promise.reject(e); };
    await expect(createSource({})).rejects.toMatchObject({ code: 'BAD_REQUEST' });
  });
});
