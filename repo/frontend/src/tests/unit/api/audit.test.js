import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import * as api from '@/api/audit.js';

describe('api/audit.js — full coverage', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getLogs', async () => { ok({ data: [] }); await api.getLogs({ page: 1 }); expect(captured.url).toBe('/audit/logs'); });
  it('getLog', async () => { ok({ id: 1 }); await api.getLog(5); expect(captured.url).toBe('/audit/logs/5'); });
  it('searchLogs', async () => { ok({ data: [] }); await api.searchLogs({ action_type: 'LOGIN' }); expect(captured.url).toBe('/audit/logs/search'); });
  it('getDeleteRequests', async () => { ok({ data: [] }); await api.getDeleteRequests(); expect(captured.url).toBe('/audit/delete-requests'); });
  it('createDeleteRequest', async () => { ok({ id: 1 }); await api.createDeleteRequest({ reason: 'cleanup' }); expect(captured.method).toBe('post'); });
  it('getDeleteRequest', async () => { ok({ id: 1 }); await api.getDeleteRequest(3); expect(captured.url).toBe('/audit/delete-requests/3'); });
  it('approveDeleteRequest', async () => { ok({}); await api.approveDeleteRequest(3); expect(captured.url).toBe('/audit/delete-requests/3/approve'); expect(captured.method).toBe('post'); });
  it('executeDeleteRequest', async () => { ok({}); await api.executeDeleteRequest(3); expect(captured.url).toBe('/audit/delete-requests/3/execute'); expect(captured.method).toBe('post'); });
});
