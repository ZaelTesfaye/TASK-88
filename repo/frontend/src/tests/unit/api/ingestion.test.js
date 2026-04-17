import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import client from '@/api/client.js';
import * as api from '@/api/ingestion.js';

describe('api/ingestion.js — full coverage', () => {
  let orig, captured;
  beforeEach(() => { orig = client.defaults.adapter; });
  afterEach(() => { client.defaults.adapter = orig; });
  function ok(d = {}) { client.defaults.adapter = (c) => { captured = c; return Promise.resolve({ data: d, status: 200, headers: {}, config: c }); }; }

  it('getSources', async () => { ok({ items: [] }); await api.getSources(); expect(captured.url).toBe('/ingestion/sources'); });
  it('createSource', async () => { ok({ id: 1 }); await api.createSource({ name: 'S' }); expect(captured.method).toBe('post'); });
  it('getSource', async () => { ok({ id: 3 }); await api.getSource(3); expect(captured.url).toBe('/ingestion/sources/3'); });
  it('updateSource', async () => { ok(); await api.updateSource(3, { name: 'X' }); expect(captured.method).toBe('put'); });
  it('deleteSource', async () => { ok(); await api.deleteSource(3); expect(captured.method).toBe('delete'); });
  it('getJobs', async () => { ok({ items: [] }); await api.getJobs({ page: 1 }); expect(captured.url).toBe('/ingestion/jobs'); });
  it('createJob', async () => { ok({ id: 1 }); await api.createJob({ import_source_id: 1 }); expect(captured.method).toBe('post'); });
  it('getJob', async () => { ok({ id: 1 }); await api.getJob(1); expect(captured.url).toBe('/ingestion/jobs/1'); });
  it('retryJob', async () => { ok(); await api.retryJob(1); expect(captured.url).toBe('/ingestion/jobs/1/retry'); expect(captured.method).toBe('post'); });
  it('acknowledgeJob', async () => { ok(); await api.acknowledgeJob(1, { acknowledged_reason: 'done' }); expect(captured.url).toBe('/ingestion/jobs/1/acknowledge'); expect(captured.method).toBe('post'); });
  it('getCheckpoints', async () => { ok({ checkpoints: [] }); await api.getCheckpoints(1); expect(captured.url).toBe('/ingestion/jobs/1/checkpoints'); });
  it('getFailures', async () => { ok({ failures: [] }); await api.getFailures(1, { page: 1 }); expect(captured.url).toBe('/ingestion/jobs/1/failures'); });
  it('getConnectorHealth', async () => { ok({ healthy: true }); await api.getConnectorHealth(1); expect(captured.url).toContain('/ingestion/connectors/1'); });
  it('getCapabilities', async () => { ok({ capabilities: [] }); await api.getCapabilities(1); expect(captured.url).toContain('/ingestion/connectors/1'); });
  it('runJob', async () => { ok({ id: 1 }); await api.runJob(1, { foo: 'bar' }); expect(captured.method).toBe('post'); });
});
