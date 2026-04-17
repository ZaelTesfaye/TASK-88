import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import { getOrgTree, createNode, updateNode, deleteNode, switchContext } from '@/api/org.js';

describe('api/org.js', () => {
  let originalAdapter, captured;
  beforeEach(() => { originalAdapter = client.defaults.adapter; captured = null; });
  afterEach(() => { client.defaults.adapter = originalAdapter; });

  function mockOK(data = {}) {
    client.defaults.adapter = (config) => { captured = config; return Promise.resolve({ data, status: 200, headers: {}, config }); };
  }
  function mock404() {
    client.defaults.adapter = (config) => {
      const e = new Error('Not Found'); e.response = { status: 404, data: { code: 'NOT_FOUND', message: 'not found' }, headers: {}, config };
      return Promise.reject(e);
    };
  }

  it('getOrgTree sends GET /org/tree', async () => { mockOK({ data: [] }); await getOrgTree(); expect(captured.url).toBe('/org/tree'); });
  it('createNode sends POST /org/nodes with parent_id', async () => {
    mockOK({ data: { id: 2 } });
    await createNode(1, { name: 'Child', level_code: 'city', level_label: 'City' });
    expect(captured.method).toBe('post');
    expect(captured.url).toBe('/org/nodes');
    const body = JSON.parse(captured.data);
    expect(body.parent_id).toBe(1);
    expect(body.name).toBe('Child');
  });
  it('updateNode sends PUT /org/nodes/:id', async () => { mockOK(); await updateNode(5, { name: 'X' }); expect(captured.url).toBe('/org/nodes/5'); expect(captured.method).toBe('put'); });
  it('deleteNode sends DELETE /org/nodes/:id', async () => { mockOK(); await deleteNode(5); expect(captured.url).toBe('/org/nodes/5'); expect(captured.method).toBe('delete'); });
  it('switchContext sends POST /context/switch with node_id', async () => {
    mockOK(); await switchContext(3);
    expect(captured.url).toBe('/context/switch');
    expect(JSON.parse(captured.data).node_id).toBe(3);
  });
  it('getOrgTree propagates 404', async () => { mock404(); await expect(getOrgTree()).rejects.toMatchObject({ code: 'NOT_FOUND' }); });
});
