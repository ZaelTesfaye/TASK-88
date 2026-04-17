import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import { getLogs, searchLogs, getDeleteRequests, createDeleteRequest } from '@/api/audit.js';

describe('api/audit.js', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getLogs sends GET /audit/logs with params', async () => { ok({ data: [] }); await getLogs({ page: 1 }); expect(captured.url).toBe('/audit/logs'); expect(captured.params.page).toBe(1); });
  it('searchLogs sends GET /audit/logs/search with params', async () => { ok({ data: [] }); await searchLogs({ action_type: 'LOGIN' }); expect(captured.url).toBe('/audit/logs/search'); expect(captured.params.action_type).toBe('LOGIN'); });
  it('getDeleteRequests sends GET /audit/delete-requests', async () => { ok({ data: [] }); await getDeleteRequests(); expect(captured.url).toBe('/audit/delete-requests'); });
  it('createDeleteRequest sends POST /audit/delete-requests', async () => { ok({ id: 1, state: 'pending' }); await createDeleteRequest({ reason: 'cleanup', target_type: 'audit_log' }); expect(captured.method).toBe('post'); expect(captured.url).toBe('/audit/delete-requests'); });
});
